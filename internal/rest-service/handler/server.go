package handler

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/konorlevich/test_task_s3/internal/rest-service/database"
	"github.com/konorlevich/test_task_s3/internal/rest-service/handler/middleware"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type ServerRegistry interface {
	AddServer(name, port string) (uuid.UUID, error)
}

type StorageRepository interface {
	ServerRegistry
	GetLeastLoadedServers(num int) ([]*database.Server, error)
	SaveFile(user, dir, name string) (uuid.UUID, error)
	GetFile(username, dir, name string) (*database.File, error)
	RemoveFile(username, dir, name string) error
	SaveChunk(file uuid.UUID, server uuid.UUID, number uint) (uuid.UUID, error)
}

func NewHandler(storageRepository StorageRepository, chunkNum int) *http.ServeMux {
	handler := http.NewServeMux()

	handler.Handle("GET /object/{dir}/{name}", middleware.CheckAuth(http.HandlerFunc(getFileHandler(storageRepository))))
	handler.Handle("POST /object/{dir}/{name}", middleware.CheckAuth(http.HandlerFunc(saveFile(storageRepository, chunkNum))))

	handler.HandleFunc("POST /storage/register", RegisterStorage(storageRepository))
	return handler
}

func getHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

func RegisterStorage(repository ServerRegistry) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil || !r.PostForm.Has("hostname") || !r.PostForm.Has("port") {
			http.Error(rw, "Can't read request data", http.StatusNotAcceptable)

			return
		}
		hostname := r.PostForm.Get("hostname")
		port := r.PostForm.Get("port")
		if _, err := repository.AddServer(hostname, port); err != nil {
			http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		log.Printf("storage service added - %s:%s", hostname, port)
		rw.WriteHeader(200)
		_, _ = rw.Write([]byte("added " + hostname))
	}
}

func getFileHandler(storage StorageRepository) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		rd, err := newRequestData(r, log.NewEntry(log.New()))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		l := log.New().WithFields(log.Fields{
			fieldNameUsername: rd.username,
			fieldNameDir:      rd.dir,
			fieldNameFileName: rd.filename,
		})

		file, err := storage.GetFile(rd.username, rd.dir, rd.filename)
		if err != nil || len(file.Chunks) == 0 {
			l.WithError(err).Error("can't get file from db")
			http.NotFound(rw, r)
			return
		}
		l.WithField("fileId", file.ID)
		b, err := getFile(file, l)
		if err != nil || len(file.Chunks) == 0 {
			l.WithError(err).Error("can't assemble the file")
			http.Error(rw, "can't assemble the file", http.StatusInternalServerError)

			return
		}

		if _, err := io.Copy(rw, b); err != nil {
			l.WithError(err).Error("can't return the file")
			http.Error(rw, "can't return the file", http.StatusInternalServerError)
		}
		l.Info("file sent")
	}
}

func saveFile(storage StorageRepository, chunkNum int) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		rd, err := newRequestData(r, log.NewEntry(log.New()))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		l := log.New().WithFields(log.Fields{
			fieldNameUsername: rd.username,
			fieldNameDir:      rd.dir,
			fieldNameFileName: rd.filename,
			"chunk_num":       chunkNum,
			"file_size":       rd.file.header.Size,
		})

		servers, err := storage.GetLeastLoadedServers(chunkNum)
		if err != nil {
			l.WithError(err).Errorf("can't get servers")
			http.Error(rw, "something went wrong, please try later", http.StatusInternalServerError)
			return
		}
		if len(servers) != chunkNum {
			l.WithField("server_num", len(servers)).Error("unexpected server count")
			http.Error(rw, "something went wrong, please try later", http.StatusInternalServerError)
			return
		}
		fileId, err := storage.SaveFile(rd.username, rd.dir, rd.filename)
		if err != nil {
			l.WithError(err).Errorf("can't save file")
			http.Error(rw, "something went wrong, please try later", http.StatusInternalServerError)
			return
		}
		chunkSize := rd.file.header.Size / int64(chunkNum)
		chunkTail := rd.file.header.Size % int64(chunkNum)
		eg := &errgroup.Group{}
		eg.SetLimit(len(servers))
		for i := 0; i < len(servers); i++ {
			body := &bytes.Buffer{}

			writer := multipart.NewWriter(body)

			formFile, err := writer.CreateFormFile("chunk", fmt.Sprintf("%d", i))
			if err != nil {
				l.WithError(err).Errorf("can't create a file field")
				http.Error(rw, "can't create a file field in multipart request body", http.StatusInternalServerError)

				return

			}

			var chunkLen int64
			if i < len(servers)-1 {
				chunkLen = chunkSize
			} else {
				chunkLen = chunkSize + chunkTail
			}

			if n, err := io.CopyN(formFile, rd.file.f, chunkLen); err != nil && errors.Is(err, io.EOF) {
				l.WithError(err).Errorf("can't read file chunk")
				http.Error(rw, "can't read the file, try again", http.StatusInternalServerError)

				return
			} else {
				l.WithField("size", n).Debug("chunk file written")
			}

			_ = writer.Close()

			c := getHTTPClient()
			server := servers[i]

			eg.Go(func() error {
				url := fmt.Sprintf("http://%s:%s/object/%s/%s", server.Name, server.Port, rd.username, fileId)
				req, err := http.NewRequest("POST", url, body)
				if err != nil {
					return fmt.Errorf("can't prepare request to %s: %w", server.Name, err)
				}
				req.Header.Add("Content-Type", writer.FormDataContentType())

				res, err := c.Do(req)
				if err != nil {
					return err
				}
				defer res.Body.Close()

				if res.StatusCode != http.StatusOK {
					body, err := io.ReadAll(res.Body)
					if err != nil {
						return fmt.Errorf("cant' read response body: %w", err)
					}
					return fmt.Errorf("can't send file to %s: %s", server.Name, body)
				}

				_, err = storage.SaveChunk(fileId, server.ID, uint(i))

				return err
			})

			if err := eg.Wait(); err != nil {
				l.WithError(err).Errorf("can't save file")

				if err = storage.RemoveFile(rd.username, rd.dir, rd.filename); err != nil {
					l.WithError(err).Errorf("can't remove chunks")
				}
				http.Error(rw, "something went wrong, please try later", http.StatusInternalServerError)
				return
			}

		}

		l.Info("file saved")
		_, _ = rw.Write([]byte("file saved"))
	}
}

func getFile(f *database.File, l *log.Entry) (io.Reader, error) {
	eg := &errgroup.Group{}
	eg.SetLimit(len(f.Chunks))
	chunkReaders := make([]io.Reader, len(f.Chunks))
	for _, chunk := range f.Chunks {
		c := getHTTPClient()
		eg.Go(func() error {
			url := fmt.Sprintf(
				"http://%s:%s/object/%s/%s/%d",
				chunk.Server.Name, chunk.Server.Port, f.User, f.ID, chunk.Number)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("can't prepare request to %s: %w", chunk.Server.Name, err)
			}

			res, err := c.Do(req)
			if err != nil {
				return fmt.Errorf("can't get chunk from %s: %w", chunk.Server.Name, err)
			}
			defer res.Body.Close()
			c := &bytes.Buffer{}

			if _, err := io.Copy(c, res.Body); err != nil {
				return fmt.Errorf("can't read chunk from %s: %w", chunk.Server.Name, err)
			}
			chunkReaders[chunk.Number] = c

			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		l.WithError(err).Error("can't get chunks from storage")

		return nil, err
	}
	return io.MultiReader(chunkReaders...), nil
}
