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
	expectedError  error
	verifyResponse func(t *testing.T, rd *requestData, err error)
}

// Helper function to create a multipart request with optional file content
func createMultipartRequest(
	includeFile bool,
	fileContent, authorized string,
	pathValues map[string]string) *http.Request {
	var buffer bytes.Buffer
	mw := multipart.NewWriter(&buffer)

	if includeFile {
		fw, err := mw.CreateFormFile("file", "testfile.txt")
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
	if authorized != "" {
		req.SetBasicAuth(authorized, "")
	}
	return req
}

func createGetRequest(url, authorized string, pathVals map[string]string) *http.Request {
	r := httptest.NewRequest("GET", url, nil)
	for name, val := range pathVals {
		r.SetPathValue(name, val)
	}
	if authorized != "" {
		r.SetBasicAuth(authorized, "")
	}
	return r

}

func TestNewRequestData(t *testing.T) {
	tests := []testCase{
		{
			description:   "Successful POST request with file chunk, no path vals",
			request:       createMultipartRequest(true, "file content", "username", map[string]string{}),
			expectedError: nil,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				assert.Equal(t, rd.username, "username")
				assert.Equal(t, rd.filename, "")
			},
		},
		{
			description: "Successful POST request with file, with path vals",
			request: createMultipartRequest(true, "file content", "username1", map[string]string{
				fieldNameUsername: "username1",
				fieldNameDir:      "dir1",
				fieldNameFileName: "file1",
			}),
			expectedError: nil,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if rd == nil || rd.file == nil {
					t.Errorf("Expected file to be present, got nil")
				}
				assert.Equal(t, "username1", rd.username)
				assert.Equal(t, "file1", rd.filename)
			},
		},
		{
			description: "Get with path vals and auth",
			request: createGetRequest("http://example.com/upload", "username2", map[string]string{
				fieldNameUsername: "username2",
				fieldNameDir:      "dir2",
				fieldNameFileName: "file2",
			}),
			expectedError: nil,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if err != nil {

					t.Errorf("Unexpected error %v", err)
				}
				assert.Equal(t, "username2", rd.username)
				assert.Equal(t, "dir2", rd.dir)
				assert.Equal(t, "file2", rd.filename)
			},
		},
		{
			description:   "Get without path vals and auth",
			request:       createGetRequest("http://example.com/upload", "", map[string]string{}),
			expectedError: nil,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
				if err != nil {

					t.Errorf("Unexpected error %v", err)
				}
				assert.Equal(t, "", rd.username)
				assert.Equal(t, "", rd.dir)
				assert.Equal(t, "", rd.filename)
			},
		},
		{
			description:   "Failure to parse multipart form",
			request:       httptest.NewRequest("POST", "http://example.com/upload", strings.NewReader("bad content")),
			expectedError: errCantParseForm,
			verifyResponse: func(t *testing.T, rd *requestData, err error) {
			},
		},
		{
			description:   "Failure when no file provided",
			request:       createMultipartRequest(false, "", "username", map[string]string{}),
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
