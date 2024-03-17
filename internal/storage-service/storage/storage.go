package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

var (
	ErrCantCreateChunkFile = errors.New("can't create chunk file")
	ErrCantWriteChunkFile  = errors.New("can't write chunk file")
	ErrCantCreateChunkDir  = errors.New("can't create chunk storage dir")

	ErrCantFindChunk = errors.New("can't find the chunk")
	ErrCantReadChunk = errors.New("can't read the chunk file")
)

type Storage struct {
	path string
	l    *log.Entry
}

func (s *Storage) GetFile(p string) ([]byte, error) {
	chunkFilePath := path.Join(s.path, p)

	if _, err := os.Stat(chunkFilePath); err != nil {
		s.l.WithError(err).Error(ErrCantFindChunk)
		return nil, ErrCantFindChunk
	}
	f, err := os.ReadFile(chunkFilePath)
	if err != nil {
		s.l.WithError(err).Error(ErrCantReadChunk)
		return nil, ErrCantReadChunk
	}
	return f, nil
}

func (s *Storage) SaveFile(p string, file io.ReadCloser) error {
	defer func(file io.ReadCloser) {
		if err := file.Close(); err != nil {
			s.l.WithError(err).Error("can't close temp file")
		}
	}(file)

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
	defer func(chunk *os.File) {
		if err := chunk.Close(); err != nil {
			log.WithError(err).Error("can't close chunk file")
		}
	}(chunk)

	if _, err := io.Copy(chunk, file); err != nil {
		log.WithError(err).Error(ErrCantWriteChunkFile)
		return ErrCantWriteChunkFile
	}
	return nil
}

func NewStorage(basePath string, l *log.Entry) (*Storage, error) {
	storagePath := path.Join(basePath, "chunks")
	if err := os.MkdirAll(storagePath, fs.ModePerm); err != nil {
		return nil, fmt.Errorf("can't create chunk storage dir: %w", err)
	}
	return &Storage{path: storagePath, l: l.WithField("storage_base_path", storagePath)}, nil
}
