package handler

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"
)

func getLogger() *log.Entry {
	logger := log.New()
	logger.SetLevel(log.FatalLevel)
	return logger.WithField("in_test", true)
}

type testCase struct {
	description    string
	request        *http.Request
	logger         *log.Entry
	expectedError  error
	verifyResponse func(t *testing.T, rd *requestData, err error)
}

// Helper function to create a multipart request with optional file content
func createMultipartRequest(includeFile bool, fileContent string, pathValues map[string]string) *http.Request {
	var buffer bytes.Buffer
	mw := multipart.NewWriter(&buffer)

	if includeFile {
		fw, err := mw.CreateFormFile("chunk", "testfile.txt")
		if err != nil {
			panic(err)
		}
		_, err = fw.Write([]byte(fileContent))
		if err != nil {
			panic(err)
		}
	}
	mw.Close()

	req := httptest.NewRequest("POST", "http://example.com/upload", &buffer)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	for name, val := range pathValues {
		req.SetPathValue(name, val)
	}
	return req
}

func createGetRequest(url string, pathVals map[string]string) *http.Request {
	r := httptest.NewRequest("GET", "http://example.com/upload", nil)
	for name, val := range pathVals {
		r.SetPathValue(name, val)
	}
	return r

}

func TestNewRequestData(t *testing.T) {
	tests := []testCase{
		{
			description: "Successful POST request with file chunk,  path vals",
			request: createMultipartRequest(
				true,
				"file content",
				map[string]string{}),
			expectedError: nil,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if rd == nil || rd.file == nil {
					t.Errorf("Expected file to be present, got nil")
				}
				assert.Equal(t, rd.username, "")
				assert.Equal(t, rd.fileId, "")
			},
		},
		{
			description: "Successful POST request with file chunk, with path vals",
			request: createMultipartRequest(
				true,
				"file content",
				map[string]string{
					fieldNameUsername: "username1",
					fieldNameFileId:   "file1",
					fieldNameChunkId:  "testfile.txt",
				}),
			expectedError: nil,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if rd == nil || rd.file == nil {
					t.Errorf("Expected file to be present, got nil")
				}
				assert.Equal(t, rd.username, "username1")
				assert.Equal(t, rd.fileId, "file1")
			},
		},
		{
			description: "Get with path vals",
			request: createGetRequest(
				"http://example.com/upload",
				map[string]string{
					fieldNameUsername: "username2",
					fieldNameFileId:   "file2",
					fieldNameChunkId:  "chunk2",
				}),
			expectedError: nil,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if err != nil {

					t.Errorf("Unexpected error %v", err)
				}
				assert.Equal(t, rd.username, "username2")
				assert.Equal(t, rd.fileId, "file2")
				assert.Equal(t, rd.chunkId, "chunk2")
			},
		},
		{
			description: "Get without path vals",
			request: createGetRequest(
				"http://example.com/upload",
				map[string]string{}),
			expectedError: nil,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if err != nil {

					t.Errorf("Unexpected error %v", err)
				}
				assert.Equal(t, rd.username, "")
				assert.Equal(t, rd.fileId, "")
				assert.Equal(t, rd.chunkId, "")
			},
		},
		{
			description:   "Failure to parse multipart form",
			request:       httptest.NewRequest("POST", "http://example.com/upload", strings.NewReader("bad content")),
			expectedError: errCantParseForm,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if !errors.Is(err, errCantParseForm) {
					t.Errorf("Expected error %v, got %v", errCantParseForm, err)
				}
			},
		},
		{
			description:   "Failure when no file chunk provided",
			request:       createMultipartRequest(false, "", map[string]string{}),
			expectedError: errNoFile,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if !errors.Is(err, errNoFile) {
					t.Errorf("Expected error %v, got %v", errNoFile, err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			rd, err := newRequestData(tc.request, getLogger())
			if !errors.Is(err, tc.expectedError) {
				t.Fatalf("Expected error %v, got %v", tc.expectedError, err)
			}
			tc.verifyResponse(t, rd, err)
		})
	}
}
