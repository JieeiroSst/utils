package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClientCircuitBreakerProxy_Get(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		expectError    bool
	}{
		{
			name:           "successful GET request",
			serverResponse: `{"message": "success"}`,
			serverStatus:   http.StatusOK,
			expectError:    false,
		},
		{
			name:           "server error",
			serverResponse: `{"message": "error"}`,
			serverStatus:   http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			cb := NewClientCircuitBreakerProxy()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			data, err := cb.Get(ctx, server.URL)
			assert.NoError(t, err)
			assert.Equal(t, []byte(tt.serverResponse), data)
		})
	}
}

func TestClientCircuitBreakerProxy_Post(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		expectError    bool
		method         string
		requestBody    []byte
	}{
		{
			name:           "successful POST request",
			serverResponse: `{"message": "success"}`,
			serverStatus:   http.StatusOK,
			expectError:    false,
			method:         http.MethodPost,
			requestBody:    []byte(`{"data": "test"}`),
		},
		{
			name:           "server error on POST request",
			serverResponse: `{"message": "error"}`,
			serverStatus:   http.StatusInternalServerError,
			expectError:    true,
			method:         http.MethodPost,
			requestBody:    []byte(`{"data": "test"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			cb := NewClientCircuitBreakerProxy()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			data, err := cb.Post(ctx, server.URL, tt.method, tt.requestBody)
			assert.NoError(t, err)
			assert.Equal(t, []byte(tt.serverResponse), data)
		})
	}
}
