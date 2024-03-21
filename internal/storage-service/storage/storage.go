package storage

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

var (
	ErrCantCreateStorage = errors.New("can't create chunk storage dir")

	ErrNothingToSave       = errors.New("empty input file")
	ErrCantCreateChunkFile = errors.New("can't create chunk file")
	ErrCantWriteChunkFile  = errors.New("can't write chunk file")
	ErrCantCloseChunkFile  = errors.New("can't close chunk file")
	ErrCantCreateChunkDir  = errors.New("can't create chunk storage dir")

	ErrIsNotAFile    = errors.New("can't find the chunk")
	ErrCantFindChunk = errors.New("can't find the chunk")
	ErrCantReadChunk = errors.New("can't read the chunk file")
)

type Storage struct {
	path string
	l    *log.Entry
}

func (s *Storage) GetFile(p string) (io.Reader, error) {
	chunkFilePath := path.Join(s.path, p)

	if fInfo, err := os.Stat(chunkFilePath); err != nil {
		s.l.WithError(err).Error(ErrCantFindChunk)
		return nil, ErrCantFindChunk
	} else if !fInfo.Mode().IsRegular() {
		return nil, ErrIsNotAFile
	}

	f, err := os.OpenFile(chunkFilePath, os.O_RDONLY, 0666)
	if err != nil {
		s.l.WithError(err).Error(ErrCantReadChunk)
		return nil, ErrCantReadChunk
	}
	return f, nil
}

func (s *Storage) SaveFile(p string, file io.Reader) error {
	if file == nil {
		return ErrNothingToSave
	}

	chunkFilePath := path.Join(s.path, p)
	chunkDir := path.Dir(chunkFilePath)
	if err := os.MkdirAll(chunkDir, fs.ModePerm); err != nil {
		s.l.
			WithField("chunk_dir", chunkDir).
			WithError(err).
			Error(ErrCantCreateChunkDir)
		return ErrCantCreateChunkDir
	}

	chunk, err := os.OpenFile(chunkFilePath, os.O_WRONLY|os.O_CREATE, fs.ModePerm)
	if err != nil {
		s.l.WithField("chunk_path", chunkFilePath).WithError(err).Error(ErrCantCreateChunkFile)
		return ErrCantCreateChunkFile
	}

	if n, err := io.Copy(chunk, file); err != nil {
		s.l.WithError(err).Error(ErrCantWriteChunkFile)
		return ErrCantWriteChunkFile
	} else {
		s.l.WithField("size", n).Debug("chunk file written")
	}
	if err = chunk.Close(); err != nil {
		s.l.WithError(err).Error(ErrCantCloseChunkFile)
		return ErrCantCloseChunkFile
	}
	return nil
}

func NewStorage(basePath string, l *log.Entry) (*Storage, error) {
	storagePath := path.Join(basePath, "chunks")
	if err := os.MkdirAll(storagePath, fs.ModePerm); err != nil {
		l.WithError(err).Error(ErrCantCreateStorage)
		return nil, ErrCantCreateStorage
	}
	return &Storage{path: storagePath, l: l.WithField("storage_base_path", storagePath)}, nil
}
