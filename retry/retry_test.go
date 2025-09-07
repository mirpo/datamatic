package retry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry_PermanentError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := Config{Enabled: true, MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffMultiplier: 2.0}

	err := Do(context.Background(), cfg, func() error {
		resp, err := http.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return &openai.RequestError{HTTPStatusCode: resp.StatusCode}
		}
		return nil
	}, ShouldRetryHTTPError)

	assert.Error(t, err)
	assert.Equal(t, 1, callCount, "Should not retry on permanent 404 error")
}

func TestRetry_RetryableError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{Enabled: true, MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffMultiplier: 2.0}

	err := Do(context.Background(), cfg, func() error {
		resp, err := http.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return &openai.RequestError{HTTPStatusCode: resp.StatusCode}
		}
		return nil
	}, ShouldRetryHTTPError)

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount, "Should retry twice before succeeding on third attempt")
}

func TestRetry_RateLimitError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{Enabled: true, MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffMultiplier: 2.0}

	err := Do(context.Background(), cfg, func() error {
		resp, err := http.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return &openai.RequestError{HTTPStatusCode: resp.StatusCode}
		}
		return nil
	}, ShouldRetryHTTPError)

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "Should retry once before succeeding on second attempt")
}

func TestRetry_RetryDisabled(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := Config{Enabled: false, MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffMultiplier: 2.0}

	err := Do(context.Background(), cfg, func() error {
		resp, err := http.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return &openai.RequestError{HTTPStatusCode: resp.StatusCode}
		}
		return nil
	}, ShouldRetryHTTPError)

	assert.Error(t, err)
	assert.Equal(t, 1, callCount, "Should not retry when retry is disabled")
}

func TestRetry_MaxRetriesExceeded(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := Config{Enabled: true, MaxAttempts: 2, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffMultiplier: 2.0}

	err := Do(context.Background(), cfg, func() error {
		resp, err := http.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return &openai.RequestError{HTTPStatusCode: resp.StatusCode}
		}
		return nil
	}, ShouldRetryHTTPError)

	assert.Error(t, err)
	assert.Equal(t, 2, callCount, "Should retry only up to max attempts")
}

func TestRetry_BackoffTiming(t *testing.T) {
	callCount := 0
	callTimes := make([]time.Time, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		callTimes = append(callTimes, time.Now())

		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{Enabled: true, MaxAttempts: 3, InitialDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second, BackoffMultiplier: 2.0}

	err := Do(context.Background(), cfg, func() error {
		resp, err := http.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return &openai.RequestError{HTTPStatusCode: resp.StatusCode}
		}
		return nil
	}, ShouldRetryHTTPError)

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
	require.Len(t, callTimes, 3)

	delay1 := callTimes[1].Sub(callTimes[0])
	delay2 := callTimes[2].Sub(callTimes[1])

	assert.GreaterOrEqual(t, delay1.Milliseconds(), int64(80), "First retry should have delay")
	assert.GreaterOrEqual(t, delay2.Milliseconds(), int64(160), "Second retry should have exponential backoff")
}

func TestShouldRetryHTTPError_PermanentErrors(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{400, false}, // Bad Request
		{401, false}, // Unauthorized
		{403, false}, // Forbidden
		{404, false}, // Not Found
		{422, false}, // Unprocessable Entity
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			err := &openai.RequestError{HTTPStatusCode: tt.statusCode}
			result := ShouldRetryHTTPError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldRetryHTTPError_RetryableErrors(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{429, true}, // Too Many Requests
		{500, true}, // Internal Server Error
		{502, true}, // Bad Gateway
		{503, true}, // Service Unavailable
		{504, true}, // Gateway Timeout
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			err := &openai.RequestError{HTTPStatusCode: tt.statusCode}
			result := ShouldRetryHTTPError(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldRetryHTTPError_NilError(t *testing.T) {
	result := ShouldRetryHTTPError(nil)
	assert.False(t, result)
}
