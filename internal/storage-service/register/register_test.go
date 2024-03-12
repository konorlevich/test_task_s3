package register

import (
	"github.com/google/go-cmp/cmp"
	"github.com/konorlevich/test_task_s3/internal/rest-service/handler"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

func TestRegister(t *testing.T) {

	type data struct {
		Hostname string
		Port     string
	}
	received := &sync.Map{}

	testHandler := http.NewServeMux()
	testHandler.HandleFunc("/path", handler.RegisterStorage(received))
	server := httptest.NewServer(testHandler)
	defer server.Close()

	testUrl, _ := url.Parse(server.URL)
	testUrl.Path = "/path"
	unexistingUrl, _ := url.Parse("http://localhost:432342234/test")

	tests := []struct {
		name     string
		url      *url.URL
		sendData data
		wantErr  bool
		wantData data
	}{
		{name: "empty", wantErr: true},
		{name: "empty port",
			sendData: data{
				Hostname: "somename",
			},
			url:      testUrl,
			wantErr:  true,
			wantData: data{Hostname: "somename"},
		},
		{name: "empty username",
			sendData: data{
				Port: "8080",
			},
			url:      testUrl,
			wantErr:  true,
			wantData: data{Port: "8080"},
		},
		{name: "empty url",
			sendData: data{
				Port:     "8080",
				Hostname: "somename",
			},
			wantErr: true,
		},
		{name: "unexisting url",
			sendData: data{
				Port:     "8080",
				Hostname: "somename",
			},
			url:     unexistingUrl,
			wantErr: true,
		},
		{name: "valid",
			sendData: data{
				Port:     "8080",
				Hostname: "somename",
			},
			url:     testUrl,
			wantErr: false,
			wantData: data{
				Port:     "8080",
				Hostname: "somename",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("check error", func(t *testing.T) {
				if err := Register(tt.url, tt.sendData.Hostname, tt.sendData.Port); (err != nil) != tt.wantErr {
					t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
			t.Run("data received", func(t *testing.T) {
				if tt.wantErr {
					return
				}
				if val, ok := received.Load(tt.wantData.Hostname); !ok || val != tt.wantData.Port {
					t.Errorf("data has not being received:\n%s", cmp.Diff(tt.wantData, data{Port: val.(string)}))
				}
			})
		})
	}
}
