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

	handler.Handle("GET /object/{dir}/{name}", middleware.CheckAuth(http.HandlerFunc(getFile(storageRepository))))
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

func getFile(storage StorageRepository) func(rw http.ResponseWriter, r *http.Request) {
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
		eg := &errgroup.Group{}
		eg.SetLimit(len(file.Chunks))
		chunkData := make([][]byte, len(file.Chunks))
		for _, chunk := range file.Chunks {
			c := getHTTPClient()
			eg.Go(func() error {
				url := fmt.Sprintf(
					"http://%s:%s/object/%s/%s/%d",
					chunk.Server.Name, chunk.Server.Port, file.User, file.ID, chunk.Number)
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					return fmt.Errorf("can't prepare request to %s: %w", chunk.Server.Name, err)
				}

				res, err := c.Do(req)
				if err != nil {
					return fmt.Errorf("can't get chunk from %s: %w", chunk.Server.Name, err)
				}
				defer res.Body.Close()
				b, err := io.ReadAll(res.Body)
				if err != nil {
					return fmt.Errorf("can't read chunk from %s: %w", chunk.Server.Name, err)
				}
				chunkData[chunk.Number] = b
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			l.WithError(err).Error("can't get chunks from storage")
			http.Error(rw, "can't get chunks from storage", http.StatusInternalServerError)

			return
		}

		b := &bytes.Buffer{}
		for _, chunk := range chunkData {
			if _, err := b.Write(chunk); err != nil {
				l.WithError(err).Error("can't join chunks to a file")
				http.Error(rw, "can't join chunks to a file", http.StatusInternalServerError)

				return
			}
		}

		_, _ = rw.Write(b.Bytes())
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
			var buf []byte
			if i < len(servers)-1 {
				buf = make([]byte, chunkSize)
			} else {
				buf = make([]byte, chunkSize+chunkTail)
			}
			_, err := rd.file.f.Read(buf)
			if err != nil && errors.Is(err, io.EOF) {
				l.WithError(err).Errorf("can't read file chunk")
				http.Error(rw, "can't read the file, try again", http.StatusInternalServerError)

				return
			}
			c := getHTTPClient()
			server := servers[i]
			eg.Go(func() error {

				body := &bytes.Buffer{}

				writer := multipart.NewWriter(body)

				formFile, err := writer.CreateFormFile("chunk", fmt.Sprintf("%d", i))
				if err != nil {
					return fmt.Errorf("can't create a file field in multipart request body: %w", err)
				}

				if _, err = io.Copy(formFile, bytes.NewReader(buf)); err != nil {
					return fmt.Errorf("can't add a file to multipart request body: %w", err)
				}

				_ = writer.Close()
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
