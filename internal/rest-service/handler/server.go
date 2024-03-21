package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/konorlevich/test_task_s3/internal/rest-service/database"
	"github.com/konorlevich/test_task_s3/internal/rest-service/handler/middleware"
	"github.com/konorlevich/test_task_s3/internal/rest-service/storage"
	"github.com/konorlevich/test_task_s3/internal/rest-service/storage/files"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var (
	ErrFileNotFound = errors.New("not found")
)

type ServerRegistry interface {
	AddServer(name, port string) (uuid.UUID, error)
}

type StorageRepository interface {
	ServerRegistry
	GetLeastLoadedServers(num int) ([]*database.Server, error)
	CreateFile(user, dir, name string) (uuid.UUID, error)
	GetFile(username, dir, name string) (*database.File, error)
	RemoveFile(id uuid.UUID) error
	SaveChunk(file uuid.UUID, server uuid.UUID, number uint) (uuid.UUID, error)
}

func NewHandler(storageRepository StorageRepository, chunkNum int, l *log.Entry) *http.ServeMux {
	handler := http.NewServeMux()

	handler.Handle("GET /object/{dir}/{name}", middleware.CheckAuth(http.HandlerFunc(getFileHandler(storageRepository, l))))
	handler.Handle("POST /object/{dir}/{name}", middleware.CheckAuth(http.HandlerFunc(saveFile(storageRepository, chunkNum, l))))

	handler.HandleFunc("POST /storage/register", RegisterStorage(storageRepository))
	return handler
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

func getFileHandler(repo StorageRepository, l *log.Entry) func(rw http.ResponseWriter, r *http.Request) {
	s := storage.NewServer(repo, files.NewFiles(l), l)
	return func(rw http.ResponseWriter, r *http.Request) {
		rd, err := newRequestData(r, log.NewEntry(log.New()))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		l := l.WithFields(log.Fields{
			fieldNameUsername: rd.username,
			fieldNameDir:      rd.dir,
			fieldNameFileName: rd.filename,
		})
		f, err := s.GetFile(rd.username, rd.dir, rd.filename)
		if err != nil {
			if errors.Is(err, ErrFileNotFound) {
				http.NotFound(rw, r)
				return
			}
			l.WithError(err).Error("can't get file")
			http.Error(rw, "can't get file", http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(rw, f); err != nil {
			l.WithError(err).Error("can't return the file")
			http.Error(rw, "can't return the file", http.StatusInternalServerError)
		}
		l.Info("file sent")
	}
}

func saveFile(repository StorageRepository, chunkNum int, l *log.Entry) func(rw http.ResponseWriter, r *http.Request) {
	s := storage.NewServer(repository, files.NewFiles(l), l)
	return func(rw http.ResponseWriter, r *http.Request) {
		rd, err := newRequestData(r, log.NewEntry(log.New()))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		l := l.WithFields(log.Fields{
			fieldNameUsername: rd.username,
			fieldNameDir:      rd.dir,
			fieldNameFileName: rd.filename,
			"chunk_num":       chunkNum,
			"file_size":       rd.file.header.Size,
		})

		err = s.SaveFile(rd.username, rd.dir, rd.filename, chunkNum, rd.file.header.Size, rd.file.f)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		l.Info("file saved")
		_, _ = rw.Write([]byte("file saved"))
	}
}
