package register

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"

	"github.com/google/go-cmp/cmp"

	"github.com/konorlevich/test_task_s3/internal/rest-service/handler"
)

type mockServerRegistry struct {
	saved       map[string]string
	returnError error
}

func newMockServerRegistry() *mockServerRegistry {
	return &mockServerRegistry{saved: make(map[string]string)}
}

func newMockServerRegistryReturnError() *mockServerRegistry {
	return &mockServerRegistry{
		saved:       make(map[string]string),
		returnError: errors.New("mock error"),
	}
}

func (m *mockServerRegistry) AddServer(name, port string) (uuid.UUID, error) {
	m.saved[name] = port
	return uuid.New(), m.returnError
}

func TestRegister(t *testing.T) {
	type data struct {
		Hostname string
		Port     string
	}

	unexistingUrl, _ := url.Parse("http://localhost:432342234/test")

	tests := []struct {
		name     string
		url      *url.URL
		sendData data
		registry *mockServerRegistry
		wantErr  bool
		wantData data
	}{
		{name: "empty",
			registry: newMockServerRegistry(), wantErr: true},
		{name: "empty port",
			sendData: data{
				Hostname: "somename",
			},
			registry: newMockServerRegistry(),
			wantErr:  true,
			wantData: data{Hostname: "somename"},
		},
		{name: "empty username",
			sendData: data{
				Port: "8080",
			},
			registry: newMockServerRegistry(),
			wantErr:  true,
			wantData: data{Port: "8080"},
		},
		{name: "empty url",
			sendData: data{
				Port:     "8080",
				Hostname: "somename",
			},
			url:      &url.URL{},
			registry: newMockServerRegistry(),
			wantErr:  true,
		},
		{name: "unexisting url",
			sendData: data{
				Port:     "8080",
				Hostname: "somename",
			},
			registry: newMockServerRegistry(),
			url:      unexistingUrl,
			wantErr:  true,
		},
		{name: "valid",
			sendData: data{
				Port:     "8080",
				Hostname: "somename",
			},
			registry: newMockServerRegistry(),
			wantErr:  false,
			wantData: data{
				Port:     "8080",
				Hostname: "somename",
			},
		},
		{name: "registry return error",
			sendData: data{
				Port:     "8080",
				Hostname: "somename",
			},
			registry: newMockServerRegistryReturnError(),
			wantErr:  true,
			wantData: data{
				Port:     "8080",
				Hostname: "somename",
			},
		},
	}

	for _, tt := range tests {
		testHandler := http.NewServeMux()
		testHandler.HandleFunc("/path", handler.RegisterStorage(tt.registry))
		server := httptest.NewServer(testHandler)
		defer server.Close()

		testUrl, _ := url.Parse(server.URL)
		testUrl.Path = "/path"
		if tt.url == nil {
			tt.url = testUrl
		}
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
				if val, ok := tt.registry.saved[tt.wantData.Hostname]; !ok || val != tt.wantData.Port {
					t.Errorf("data has not being received:\n%s", cmp.Diff(tt.wantData, data{Port: val}))
				}
			})
		})
	}
}
