package handler

import (
	"errors"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const (
	fieldNameUsername = "username"
	fieldNameFileId   = "file_id"
	fieldNameChunkId  = "chunk_id"
)

var (
	errCantParseForm = errors.New("can't parse request form")
	errNoFile        = errors.New("file has not been provided")
	errCantReadChunk = errors.New("can't read file from request")
)

type requestData struct {
	username string
	fileId   string
	chunkId  string
	file     io.ReadCloser
}

func newRequestData(r *http.Request, logger *log.Entry) (*requestData, error) {
	rd := &requestData{
		username: r.PathValue(fieldNameUsername),
		fileId:   r.PathValue(fieldNameFileId),
	}
	if r.Method == "GET" {
		rd.chunkId = r.PathValue(fieldNameChunkId)
		return rd, nil
	}
	l := logger.WithFields(log.Fields{
		fieldNameUsername: rd.username,
		fieldNameFileId:   rd.fileId,
	})

	//TODO: use http.MaxBytesReader instead?
	err := r.ParseMultipartForm(128 << 20) // 128Mb
	if err != nil {
		l.WithError(err).Error(errCantParseForm)
		return nil, errCantParseForm
	}
	form := r.MultipartForm

	headers, ok := form.File["chunk"]
	if !ok {

		l.WithField("field", "file").Error(errNoFile)
		return nil, errNoFile
	}
	rd.chunkId = headers[0].Filename
	l = l.WithField("chunkName", rd.chunkId)

	tmpFile, err := headers[0].Open()
	if err != nil {
		l.WithError(err).Error(errCantReadChunk)
		return nil, errCantReadChunk
	}
	rd.file = tmpFile
	return rd, nil
}
