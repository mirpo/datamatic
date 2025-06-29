package step

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptStep_RetryIntegration_PermanentError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, true, 3)
	step := createTestStep(server.URL)

	promptStep := &PromptStep{}
	err := promptStep.Run(context.Background(), cfg, step, t.TempDir())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get response from LLM")
	assert.Equal(t, 1, callCount, "Should not retry on permanent 404 error")
}

func TestPromptStep_RetryIntegration_RetryableError(t *testing.T) {
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
		response := map[string]interface{}{
			"message": map[string]interface{}{
				"content": "Test response",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, true, 3)
	step := createTestStep(server.URL)

	promptStep := &PromptStep{}
	err := promptStep.Run(context.Background(), cfg, step, t.TempDir())

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount, "Should retry twice before succeeding on third attempt")
}

func TestPromptStep_RetryIntegration_RateLimitError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		// Success on second attempt
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"message": map[string]interface{}{
				"content": "Test response",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, true, 3)
	step := createTestStep(server.URL)

	promptStep := &PromptStep{}
	err := promptStep.Run(context.Background(), cfg, step, t.TempDir())

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "Should retry once before succeeding on second attempt")
}

func TestPromptStep_RetryIntegration_RetryDisabled(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, false, 3)
	step := createTestStep(server.URL)

	promptStep := &PromptStep{}
	err := promptStep.Run(context.Background(), cfg, step, t.TempDir())

	assert.Error(t, err)
	assert.Equal(t, 1, callCount, "Should not retry when retry is disabled")
}

func TestPromptStep_RetryIntegration_MaxRetriesExceeded(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error": "service unavailable"}`))
	}))
	defer server.Close()

	cfg := createTestConfig(server.URL, true, 2)
	step := createTestStep(server.URL)

	promptStep := &PromptStep{}
	err := promptStep.Run(context.Background(), cfg, step, t.TempDir())

	assert.Error(t, err)
	assert.Equal(t, 2, callCount, "Should retry only up to max attempts")
}

func TestPromptStep_RetryIntegration_BackoffTiming(t *testing.T) {
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
		// Success on third attempt
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"message": map[string]interface{}{
				"content": "Test response",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := createTestConfigWithRetrySettings(server.URL, true, 3, 100*time.Millisecond)
	step := createTestStep(server.URL)

	promptStep := &PromptStep{}
	err := promptStep.Run(context.Background(), cfg, step, t.TempDir())

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
	require.Len(t, callTimes, 3)

	// Check that there was a delay between calls (at least 80ms to account for timing variations)
	delay1 := callTimes[1].Sub(callTimes[0])
	delay2 := callTimes[2].Sub(callTimes[1])

	assert.GreaterOrEqual(t, delay1.Milliseconds(), int64(80), "First retry should have delay")
	assert.GreaterOrEqual(t, delay2.Milliseconds(), int64(160), "Second retry should have exponential backoff")
}

// Helper functions

func createTestConfig(baseURL string, retryEnabled bool, maxAttempts int) *config.Config {
	return createTestConfigWithRetrySettings(baseURL, retryEnabled, maxAttempts, 1*time.Second)
}

func createTestConfigWithRetrySettings(baseURL string, retryEnabled bool, maxAttempts int, initialDelay time.Duration) *config.Config {
	cfg := config.NewConfig()
	cfg.RetryConfig = config.RetryConfig{
		Enabled:           retryEnabled,
		MaxAttempts:       maxAttempts,
		InitialDelay:      initialDelay,
		MaxDelay:          5 * time.Second,
		BackoffMultiplier: 2.0,
	}
	cfg.HTTPTimeout = 5
	cfg.ValidateResponse = false

	return cfg
}

func createTestStep(baseURL string) config.Step {
	tempFile, _ := os.CreateTemp("", "test_output_*.jsonl")
	tempFile.Close()

	return config.Step{
		Name:               "test_step",
		Type:               config.PromptStepType,
		Prompt:             "Test prompt",
		OutputFilename:     tempFile.Name(),
		ResolvedMaxResults: 1,
		ModelConfig: config.ModelConfig{
			ModelProvider: llm.ProviderOllama,
			ModelName:     "test-model",
			BaseURL:       baseURL,
		},
		JSONSchema: jsonl.JSONSchema{},
	}
}
