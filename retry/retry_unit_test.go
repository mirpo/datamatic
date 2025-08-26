package retry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRetryable(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"404 Not Found", &testHTTPError{statusCode: 404, message: "not found"}, false},
		{"429 Rate Limit", &testHTTPError{statusCode: 429, message: "rate limited"}, true},
		{"500 Server Error", &testHTTPError{statusCode: 500, message: "server error"}, true},
		{"502 Bad Gateway", &testHTTPError{statusCode: 502, message: "bad gateway"}, true},
		{"503 Service Unavailable", &testHTTPError{statusCode: 503, message: "unavailable"}, true},
		{"504 Gateway Timeout", &testHTTPError{statusCode: 504, message: "timeout"}, true},
		{"401 Unauthorized", &testHTTPError{statusCode: 401, message: "unauthorized"}, false},
		{"403 Forbidden", &testHTTPError{statusCode: 403, message: "forbidden"}, false},
		{"400 Bad Request", &testHTTPError{statusCode: 400, message: "bad request"}, false},
		{"422 Unprocessable", &testHTTPError{statusCode: 422, message: "unprocessable"}, false},
		{"Unknown HTTP code", &testHTTPError{statusCode: 418, message: "teapot"}, false},
		{"Network timeout", &testTimeoutError{}, true},
		{"Generic error", errors.New("generic error"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsRetryable(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExecute_Success(t *testing.T) {
	cfg := Config{MaxAttempts: 3}
	callCount := 0

	result, err := Execute(context.Background(), cfg, func() (string, error) {
		callCount++
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, callCount, "Should succeed on first attempt")
}

func TestExecute_PermanentError(t *testing.T) {
	cfg := Config{MaxAttempts: 3}
	callCount := 0

	_, err := Execute(context.Background(), cfg, func() (string, error) {
		callCount++
		return "", &testHTTPError{statusCode: 404, message: "not found"}
	})

	assert.Error(t, err)
	assert.Equal(t, 1, callCount, "Should not retry permanent errors")
}

func TestExecute_RetryableError(t *testing.T) {
	cfg := Config{MaxAttempts: 3}
	callCount := 0

	_, err := Execute(context.Background(), cfg, func() (string, error) {
		callCount++
		return "", &testHTTPError{statusCode: 500, message: "server error"}
	})

	assert.Error(t, err)
	assert.Equal(t, 3, callCount, "Should retry up to max attempts")
}

func TestExecute_EventualSuccess(t *testing.T) {
	cfg := Config{MaxAttempts: 3}
	callCount := 0

	result, err := Execute(context.Background(), cfg, func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", &testHTTPError{statusCode: 500, message: "server error"}
		}
		return "success after retries", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success after retries", result)
	assert.Equal(t, 3, callCount, "Should succeed on third attempt")
}

// Test helpers
type testTimeoutError struct{}

func (e *testTimeoutError) Error() string   { return "timeout error" }
func (e *testTimeoutError) Timeout() bool   { return true }
func (e *testTimeoutError) Temporary() bool { return true }
