package llm

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProvider(t *testing.T) {
	t.Run("returns Ollama provider", func(t *testing.T) {
		provider, err := NewProvider(ProviderConfig{ProviderType: ProviderOllama})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("returns LmStudio provider", func(t *testing.T) {
		provider, err := NewProvider(ProviderConfig{ProviderType: ProviderLmStudio})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("returns OpenAI provider", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "test-key")
		provider, err := NewProvider(ProviderConfig{ProviderType: ProviderOpenAI})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("returns error for unknown provider", func(t *testing.T) {
		provider, err := NewProvider(ProviderConfig{ProviderType: ProviderUnknown})
		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, err)) // simple check, or you could check message
		assert.Contains(t, err.Error(), "provider type 'unknown' is not supported")
	})

	t.Run("returns error for unsupported provider", func(t *testing.T) {
		provider, err := NewProvider(ProviderConfig{ProviderType: "something-else"})
		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported provider: something-else")
	})
}
