package handler

import (
	"io"
	"net/http"
	"path"

	log "github.com/sirupsen/logrus"
)

const (
	urlPatternGetChunk  = "GET /object/{username}/{file_id}/{chunk_id}"
	urlPatternSaveChunk = "POST /object/{username}/{file_id}"
)

type Storage interface {
	SaveFile(path string, file io.ReadCloser) error
	GetFile(filePath string) (io.Reader, error)
}

func NewHandler(storage Storage) *http.ServeMux {
	handler := http.NewServeMux()

	handler.HandleFunc(urlPatternGetChunk, func(rw http.ResponseWriter, r *http.Request) {
		l := log.New().WithField("client", r.RemoteAddr)
		rd, err := newRequestData(r, l)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		l.Info("file requested")

		l = l.WithFields(log.Fields{
			"username": rd.username,
			"file_id":  rd.fileId,
			"chunk_id": rd.chunkId,
		})

		chunkFilePath := path.Join(rd.username, rd.fileId, rd.chunkId)
		l = l.WithField("file_path", chunkFilePath)
		f, err := storage.GetFile(chunkFilePath)
		if err != nil {
			l.Error(err)
			http.NotFound(rw, r)
			return
		}

		if i, err := io.Copy(rw, f); err != nil {
			l.Error(err)
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		} else {
			l.WithField("size", i).Debug("chunk sent")
		}
	})

	handler.HandleFunc(urlPatternSaveChunk, func(rw http.ResponseWriter, r *http.Request) {
		l := log.New().WithField("client", r.RemoteAddr)
		rd, err := newRequestData(r, l)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		l = l.WithFields(
			log.Fields{
				fieldNameUsername: rd.username,
				fieldNameFileId:   rd.fileId,
				fieldNameChunkId:  rd.chunkId,
			})
		l.Info("file received")

		if err := storage.SaveFile(path.Join(rd.username, rd.fileId, rd.chunkId), rd.file); err != nil {
			l.WithError(err).Error("can't save file")
			http.Error(rw, "can't save file", http.StatusInternalServerError)
			return
		}

		l.Info("chunk saved")
		_, _ = rw.Write([]byte("chunk saved"))
	})

	return handler
}
