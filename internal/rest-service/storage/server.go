package storage

import (
	"errors"
	"io"
	"mime/multipart"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/konorlevich/test_task_s3/internal/rest-service/database"
	"github.com/konorlevich/test_task_s3/internal/rest-service/storage/files"
)

var (
	ErrFileNotFound = errors.New("not found")
	ErrCantGetFile  = errors.New("can't get file from db")
	ErrNoChunks     = errors.New("no chunks found for the file")

	ErrCantGetServers = errors.New("can't get servers")
	ErrCantSaveFile   = errors.New("can't save file")

	ErrSavingFailed = errors.New("file saving failed")
)

type MetaStorage interface {
	GetLeastLoadedServers(num int) ([]*database.Server, error)
	CreateFile(user, dir, name string) (uuid.UUID, error)
	GetFile(username, dir, name string) (*database.File, error)
	RemoveFile(id uuid.UUID) error
	SaveChunk(file uuid.UUID, server uuid.UUID, number uint) (uuid.UUID, error)
}

type FileStorage interface {
	SendFile(servers []files.ServerMeta, username string, file io.Reader, fileId uuid.UUID, fileSize int64) ([]uuid.UUID, error)
	GetFile(servers []files.ServerMeta, username string, fileId uuid.UUID) (io.Reader, error)
}

type Server struct {
	ms MetaStorage
	fs FileStorage
	l  *log.Entry
}

func NewServer(ms MetaStorage, fs FileStorage, l *log.Entry) *Server {
	return &Server{
		ms: ms,
		fs: fs,
		l:  l,
	}
}

func (s *Server) GetFile(username, dir, filename string) (io.Reader, error) {
	file, err := s.ms.GetFile(username, dir, filename)
	if err != nil {
		if errors.Is(err, database.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		s.l.WithError(err).Error(ErrCantGetFile)
		return nil, ErrCantGetFile
	}
	if len(file.Chunks) == 0 {
		s.l.WithError(err).Error(ErrNoChunks)
		return nil, ErrNoChunks
	}
	servers := make([]files.ServerMeta, len(file.Chunks))
	for _, chunk := range file.Chunks {
		servers[chunk.Number] = chunk.Server
	}
	return s.fs.GetFile(servers, username, file.ID)
}

func (s *Server) SaveFile(username string, dir string, filename string, chunkNum int, fileSize int64, f multipart.File) (err error) {
	servers, err := s.getServers(chunkNum)
	if err != nil {
		s.l.WithError(err).Error(ErrCantGetServers)
		return ErrCantGetServers
	}
	fileId, err := s.ms.CreateFile(username, dir, filename)
	if err != nil {
		s.l.WithError(err).Error(ErrCantSaveFile)
		return ErrCantSaveFile
	}

	var savedTo []uuid.UUID
	savedTo, err = s.fs.SendFile(servers, username, f, fileId, fileSize)
	if err != nil {
		s.l.WithError(err).Error(ErrSavingFailed)
		return ErrSavingFailed
	}
	for i, u := range savedTo {
		_, err = s.ms.SaveChunk(fileId, u, uint(i))
		if err != nil {
			_ = s.removeFile(fileId, err)
		}
	}
	return err
}

func (s *Server) getServers(num int) ([]files.ServerMeta, error) {
	serversTemp, err := s.ms.GetLeastLoadedServers(num)
	if err != nil {
		return nil, err
	}
	servers := make([]files.ServerMeta, len(serversTemp))
	for i, server := range serversTemp {
		servers[i] = server
	}
	return servers, nil
}

func (s *Server) removeFile(fileID uuid.UUID, reason error) error {
	s.l.WithError(reason).Warning("removing file")
	if err := s.ms.RemoveFile(fileID); err != nil {
		s.l.WithError(err).Errorf("can't remove chunks")
		return err
	}
	return nil
}
