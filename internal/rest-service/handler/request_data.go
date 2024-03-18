package handler

import (
	"errors"
	"mime/multipart"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const (
	fieldNameDir      = "dir"
	fieldNameUsername = "username"
	fieldNameFileName = "name"
)

var (
	errCantParseForm = errors.New("can't parse request form")
	errNoFile        = errors.New("file has not been provided")
)

type requestData struct {
	username string
	dir      string
	filename string
	file     *fileData
}

type fileData struct {
	f      multipart.File
	header *multipart.FileHeader
}

func newRequestData(r *http.Request, logger *log.Entry) (*requestData, error) {
	username, _, _ := r.BasicAuth()
	rd := &requestData{
		username: username,
		dir:      r.PathValue(fieldNameDir),
		filename: r.PathValue(fieldNameFileName),
	}
	l := logger.WithFields(log.Fields{
		fieldNameUsername: username,
		fieldNameDir:      rd.dir,
		fieldNameFileName: rd.filename,
	})
	if r.Method == "GET" {
		return rd, nil
	}

	if err := r.ParseForm(); err != nil {
		l.WithError(err).Error(errCantParseForm)
		return nil, errCantParseForm
	}

	f, fh, err := r.FormFile("file")
	if err != nil {
		l.WithError(err).Error(errNoFile)
		return nil, errNoFile
	}
	defer func(f multipart.File) {
		_ = f.Close()
	}(f)
	rd.file = &fileData{f: f, header: fh}
	return rd, nil
}
