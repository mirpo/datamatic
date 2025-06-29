package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPError_Error(t *testing.T) {
	err := &HTTPError{
		StatusCode: 404,
		Body:       []byte("not found"),
		Err:        errors.New("resource missing"),
	}
	expected := "HTTP error 404: resource missing - body: not found"
	assert.Equal(t, expected, err.Error())
}

func TestHTTPError_NotFound(t *testing.T) {
	err := &HTTPError{StatusCode: http.StatusNotFound}
	assert.True(t, err.NotFound())

	err = &HTTPError{StatusCode: http.StatusInternalServerError}
	assert.False(t, err.NotFound())
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	client := NewClient("http://example.com", "token")
	assert.Equal(t, "http://example.com", client.BaseURL)
	assert.Equal(t, "token", client.AuthToken)
	assert.Equal(t, 30*time.Second, client.HTTPClient.Timeout)
}

func TestWithTimeout(t *testing.T) {
	client := NewClient("http://example.com", "token", WithTimeout(10*time.Second))
	assert.Equal(t, 10*time.Second, client.HTTPClient.Timeout)
}

func TestClient_Post_Success(t *testing.T) {
	// Setup fake server
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		bodyBytes, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		assert.JSONEq(t, `{"foo":"bar"}`, string(bodyBytes))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"ok"}`)) //nolint:golint,errcheck
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := NewClient(server.URL, "token")

	var response struct {
		Message string `json:"message"`
	}

	err := client.Post(context.Background(), "/test", map[string]string{"foo": "bar"}, &response, http.Header{})
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.Message)
}

func TestClient_Post_HTTPError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := NewClient(server.URL, "token")

	err := client.Post(context.Background(), "/notfound", nil, nil, http.Header{})

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
		assert.Contains(t, string(httpErr.Body), "not found")
	} else {
		t.Fatalf("expected HTTPError but got %v", err)
	}
}

func TestClient_Post_InvalidJSONResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`)) //nolint:golint,errcheck
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := NewClient(server.URL, "token")

	var response struct {
		Message string `json:"message"`
	}

	err := client.Post(context.Background(), "/invalid-json", nil, &response, http.Header{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error unmarshaling response body")
}

func TestClient_Post_ErrorMarshallingRequestBody(t *testing.T) {
	client := NewClient("http://example.com", "token")

	ctx := context.Background()
	// circular reference can't be marshalled
	type BadRequest struct {
		Self *BadRequest `json:"self"`
	}
	bad := BadRequest{}
	bad.Self = &bad

	err := client.Post(ctx, "/test", bad, nil, http.Header{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error marshaling request body")
}

func TestClient_Post_ErrorJoiningURL(t *testing.T) {
	client := NewClient("http://[::1]:namedport", "token") // malformed baseURL
	ctx := context.Background()

	err := client.Post(ctx, "/path", nil, nil, http.Header{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error joining URL")
}

func TestHTTPError_IsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"429 Too Many Requests - retryable", http.StatusTooManyRequests, true},
		{"500 Internal Server Error - retryable", http.StatusInternalServerError, true},
		{"502 Bad Gateway - retryable", http.StatusBadGateway, true},
		{"503 Service Unavailable - retryable", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout - retryable", http.StatusGatewayTimeout, true},
		{"400 Bad Request - not retryable", http.StatusBadRequest, false},
		{"401 Unauthorized - not retryable", http.StatusUnauthorized, false},
		{"404 Not Found - not retryable", http.StatusNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &HTTPError{StatusCode: tt.statusCode}
			assert.Equal(t, tt.expected, err.IsRetryable())
		})
	}
}

func TestHTTPError_IsPermanent(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"400 Bad Request - permanent", http.StatusBadRequest, true},
		{"401 Unauthorized - permanent", http.StatusUnauthorized, true},
		{"403 Forbidden - permanent", http.StatusForbidden, true},
		{"404 Not Found - permanent", http.StatusNotFound, true},
		{"405 Method Not Allowed - permanent", http.StatusMethodNotAllowed, true},
		{"406 Not Acceptable - permanent", http.StatusNotAcceptable, true},
		{"409 Conflict - permanent", http.StatusConflict, true},
		{"422 Unprocessable Entity - permanent", http.StatusUnprocessableEntity, true},
		{"429 Too Many Requests - not permanent", http.StatusTooManyRequests, false},
		{"500 Internal Server Error - not permanent", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &HTTPError{StatusCode: tt.statusCode}
			assert.Equal(t, tt.expected, err.IsPermanent())
		})
	}
}

func TestHTTPError_IsRateLimited(t *testing.T) {
	err := &HTTPError{StatusCode: http.StatusTooManyRequests}
	assert.True(t, err.IsRateLimited())

	err = &HTTPError{StatusCode: http.StatusInternalServerError}
	assert.False(t, err.IsRateLimited())
}
