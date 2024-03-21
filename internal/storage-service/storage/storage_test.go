package storage

import (
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/stretchr/testify/assert"

	"github.com/google/go-cmp/cmp"

	log "github.com/sirupsen/logrus"
)

func getLogger() *log.Logger {
	l := log.New()
	l.SetLevel(log.InfoLevel)
	//l.SetLevel(log.FatalLevel)
	return l
}

func TestStorage_GetFile(t *testing.T) {
	tests := []struct {
		name    string
		p       string
		want    string
		wantErr error
	}{
		{name: "empty", wantErr: ErrIsNotAFile},
		{name: "valid file", p: "file1.txt", want: "Now you see me"},
		{name: "can't find", p: "file2.txt", wantErr: ErrCantFindChunk},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				path: "./testdata",
				l:    getLogger().WithField("test", tt.name),
			}
			gotReader, err := s.GetFile(tt.p)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotReader == nil {
				if tt.want != "" {
					t.Error("value expected, got nil")
				}
				return
			}
			gotStr, err := io.ReadAll(gotReader)
			if err != nil {
				t.Error("unexpected error")
			}
			if diff := cmp.Diff(tt.want, string(gotStr)); diff != "" {
				t.Errorf("GetFile()\n%s", diff)
			}
		})
	}
}

type readerWithError struct {
}

func (readerWithError) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestStorage_SaveFile(t *testing.T) {
	_ = os.MkdirAll("./testdata/failedDir", os.ModePerm)
	_ = os.MkdirAll("./testdata/chunks/dir", os.ModePerm)
	if f, err := os.OpenFile("./testdata/failedDir/chunks", os.O_CREATE, os.ModePerm); err != nil {
		t.Fatalf("can't create a file: %s", err)
	} else {
		f.Close()
	}
	if f, err := os.OpenFile("./testdata/chunks/failedDir", os.O_CREATE, os.ModePerm); err != nil {
		t.Fatalf("can't create a file: %s", err)
	} else {
		f.Close()
	}

	defer os.RemoveAll("./testdata/chunks")
	defer os.RemoveAll("./testdata/failedDir")
	//defer os.RemoveAll("./chunks")
	//defer func() {
	//
	//	if err := os.Remove(""); err != nil {
	//		getLogger().WithField("file", )
	//	}}()

	dummyFile := io.NopCloser(strings.NewReader("won't be needed"))
	tests := []struct {
		name        string
		storagePath string
		fileName    string
		file        io.ReadCloser
		initErr     error
		wantErr     error
		want        string
	}{
		{name: "empty", storagePath: "./testdata",
			wantErr: ErrNothingToSave,
		},

		{name: "can't init",
			storagePath: "./testdata/failedDir",
			file:        dummyFile,
			initErr:     ErrCantCreateStorage},
		{name: "failed dir",
			storagePath: "./testdata",
			fileName:    "failedDir/testFile/sdf",
			file:        dummyFile,
			wantErr:     ErrCantCreateChunkDir},
		{name: "file in failed dir",
			storagePath: "./testdata",
			fileName:    "dir",
			file:        dummyFile,
			wantErr:     ErrCantCreateChunkFile},
		{name: "can't write",
			storagePath: "./testdata",
			fileName:    "cantWrite.file",
			file:        io.NopCloser(readerWithError{}),
			wantErr:     ErrCantWriteChunkFile},
		{name: "can't write",
			storagePath: "./testdata",
			fileName:    "cantWrite.file",
			file:        io.NopCloser(readerWithError{}),
			wantErr:     ErrCantWriteChunkFile},
		{name: "success",
			storagePath: "./testdata",
			fileName:    "success.txt",
			file:        io.NopCloser(strings.NewReader("success")),
			want:        "success",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewStorage(tt.storagePath,
				getLogger().WithField("test", tt.name))
			if !errors.Is(err, tt.initErr) {
				t.Errorf("unexpected error:\n%s", cmp.Diff(tt.initErr, err, cmpopts.EquateErrors()))
			}
			if tt.initErr != nil {
				return
			}
			err = s.SaveFile(tt.fileName, tt.file)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("unexpected error:\n%s", cmp.Diff(tt.wantErr, err, cmpopts.EquateErrors()))
			}
			if tt.wantErr == nil {
				got, err := os.ReadFile(path.Join(s.path, tt.fileName))
				assert.NoError(t, err)
				assert.Equal(t, tt.want, string(got))
			}
		})
	}
}
