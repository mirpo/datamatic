package retry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_HTTPIntegration_PermanentError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer server.Close()

	cfg := Config{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond}

	_, err := Execute(context.Background(), cfg, func() (string, error) {
		resp, err := http.Get(server.URL)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", &testHTTPError{statusCode: resp.StatusCode, message: "HTTP error"}
		}
		return "success", nil
	})

	assert.Error(t, err)
	assert.Equal(t, 1, callCount, "Should not retry on permanent 404 error")
}

func TestExecute_HTTPIntegration_RetryableError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "server error"}`))
			return
		}
		// Success on third attempt
		w.WriteHeader(http.StatusOK)
		response := map[string]any{
			"result": "success",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := Config{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond}

	result, err := Execute(context.Background(), cfg, func() (string, error) {
		resp, err := http.Get(server.URL)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", &testHTTPError{statusCode: resp.StatusCode, message: "HTTP error"}
		}
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount, "Should retry twice before succeeding on third attempt")
}

func TestExecute_HTTPIntegration_BackoffTiming(t *testing.T) {
	callCount := 0
	callTimes := make([]time.Time, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		callTimes = append(callTimes, time.Now())

		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "server error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	cfg := Config{
		MaxAttempts:       3,
		InitialDelay:      50 * time.Millisecond,
		MaxDelay:          500 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	_, err := Execute(context.Background(), cfg, func() (string, error) {
		resp, err := http.Get(server.URL)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", &testHTTPError{statusCode: resp.StatusCode, message: "HTTP error"}
		}
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
	require.Len(t, callTimes, 3)

	// Check that there was a delay between calls
	delay1 := callTimes[1].Sub(callTimes[0])
	delay2 := callTimes[2].Sub(callTimes[1])

	assert.GreaterOrEqual(t, delay1.Milliseconds(), int64(40), "First retry should have delay")
	assert.GreaterOrEqual(t, delay2.Milliseconds(), int64(80), "Second retry should have exponential backoff")
}
