package files

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var (
	ErrCantGetChunks       = errors.New("can't get chunks from storage")
	ErrCantCreateFileField = errors.New("can't create a file field")
	ErrCantReadFileChunk   = errors.New("can't read file chunk")
)

type ServerMeta interface {
	GetUrl() string
	GetID() uuid.UUID
}

type requester interface {
	Get(url string) (resp *http.Response, err error)
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
}
type Files struct {
	r requester
	l *log.Entry
}

func NewFiles(l *log.Entry) *Files {
	return &Files{r: getHTTPClient(), l: l}
}

func (f *Files) GetFile(servers []ServerMeta, username string, fileId uuid.UUID) (io.Reader, error) {
	eg := &errgroup.Group{}
	eg.SetLimit(len(servers))
	chunkReaders := make([]io.Reader, len(servers))
	for i := range servers {
		chunkNumber := i
		server := servers[i]
		eg.Go(func() error {
			urlString, err := url.JoinPath(
				server.GetUrl(), "object", username, fileId.String(), fmt.Sprintf("%d", chunkNumber))
			if err != nil {
				return err
			}

			res, err := f.r.Get(urlString)
			if err != nil {
				return fmt.Errorf("can't get chunk from %s: %w", server.GetID().String(), err)
			}
			defer res.Body.Close()
			c := &bytes.Buffer{}

			if _, err = io.Copy(c, res.Body); err != nil {
				return fmt.Errorf("can't read chunk from %s: %w", server.GetID().String(), err)
			}
			chunkReaders[chunkNumber] = c

			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		f.l.WithError(err).Error(ErrCantGetChunks)

		return nil, ErrCantGetChunks
	}
	return io.MultiReader(chunkReaders...), nil
}

func (f *Files) SendFile(servers []ServerMeta, username string, file io.Reader, fileId uuid.UUID, fileSize int64) ([]uuid.UUID, error) {
	var chunkNum = int64(len(servers))
	chunkTailSize := fileSize % chunkNum
	chunkSize := fileSize / chunkNum

	var saved = make([]uuid.UUID, chunkNum)
	eg := &errgroup.Group{}
	eg.SetLimit(len(servers))
	for i := range servers {
		server := servers[i]
		saved[i] = server.GetID()

		chunkLen := chunkSize
		if i == len(servers)-1 {
			chunkLen = chunkSize + chunkTailSize
		}

		ct, r, err := f.prepareRequest(fmt.Sprintf("%d", i), file, chunkLen)
		if err != nil {
			return nil, err
		}
		eg.Go(func() error {
			urlString, err := url.JoinPath(server.GetUrl(), "object", username, fileId.String())
			if err != nil {
				f.l.
					WithFields(log.Fields{
						"server_id": server.GetID(),
						"base_url":  server.GetUrl(),
						"username":  username,
						"fileId":    fileId.String(),
					}).
					WithError(err).
					Error("can't combine url parts")
				return err
			}

			res, err := f.r.Post(urlString, ct, r)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				body, err := io.ReadAll(res.Body)
				if err != nil {
					return fmt.Errorf("cant' read response body: %w", err)
				}
				return fmt.Errorf("can't send file to server %s: %s", server.GetID(), body)
			}

			return err
		})

	}
	return saved, eg.Wait()
}

func (f *Files) prepareRequest(chunkName string, file io.Reader, chunkLen int64) (string, io.Reader, error) {
	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)

	formFile, err := writer.CreateFormFile("chunk", chunkName)
	if err != nil {
		f.l.WithError(err).Error(ErrCantCreateFileField)
		return "", nil, ErrCantCreateFileField
	}

	if n, err := io.CopyN(formFile, file, chunkLen); err != nil && errors.Is(err, io.EOF) {
		f.l.WithError(err).Error(ErrCantReadFileChunk)
		return "", nil, ErrCantReadFileChunk
	} else {
		f.l.WithField("size", n).Debug("chunk file written")
	}

	_ = writer.Close()
	return writer.FormDataContentType(), body, nil
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
