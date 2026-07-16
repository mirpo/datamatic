package llm

import (
	"context"
	"testing"
	"time"

	"github.com/mirpo/datamatic/internal/llmtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_RespectsHTTPTimeout(t *testing.T) {
	srv := llmtest.NewServer(t, "too late")
	srv.Delay = 2 * time.Second

	provider := NewOpenAIProvider(ProviderConfig{
		BaseURL:     srv.URL,
		ModelName:   "m",
		HTTPTimeout: 1, // seconds
	})

	start := time.Now()
	_, err := provider.Generate(context.Background(), GenerateRequest{UserMessage: "hi"})

	require.Error(t, err)
	assert.Less(t, time.Since(start), 2*time.Second, "must abort before server responds")
}

func TestGenerate_ZeroTimeoutMeansNoTimeout(t *testing.T) {
	srv := llmtest.NewServer(t, "ok")

	provider := NewOpenAIProvider(ProviderConfig{BaseURL: srv.URL, ModelName: "m", HTTPTimeout: 0})

	resp, err := provider.Generate(context.Background(), GenerateRequest{UserMessage: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Text)
}

func TestGenerate_TemperatureZeroIsSent(t *testing.T) {
	srv := llmtest.NewServer(t, "ok")

	temp := 0.0
	provider := NewOpenAIProvider(ProviderConfig{BaseURL: srv.URL, ModelName: "m", Temperature: &temp})

	_, err := provider.Generate(context.Background(), GenerateRequest{UserMessage: "hi"})
	require.NoError(t, err)

	req := srv.Requests()[0]
	got, present := req["temperature"]
	require.True(t, present, "temperature: 0 must be serialized, not dropped by omitempty")
	assert.InDelta(t, 0.0, got.(float64), 1e-6)
}

func TestGenerate_NilTemperatureIsNotSent(t *testing.T) {
	srv := llmtest.NewServer(t, "ok")

	provider := NewOpenAIProvider(ProviderConfig{BaseURL: srv.URL, ModelName: "m"})

	_, err := provider.Generate(context.Background(), GenerateRequest{UserMessage: "hi"})
	require.NoError(t, err)

	_, present := srv.Requests()[0]["temperature"]
	assert.False(t, present)
}
