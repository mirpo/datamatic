package llm

import (
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
		t.Setenv("OPENAI_API_KEY", "test-key")
		provider, err := NewProvider(ProviderConfig{ProviderType: ProviderOpenAI})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("returns OpenRouter provider", func(t *testing.T) {
		t.Setenv("OPENROUTER_API_KEY", "test-key")
		provider, err := NewProvider(ProviderConfig{ProviderType: ProviderOpenRouter})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("returns Gemini provider", func(t *testing.T) {
		t.Setenv("GEMINI_API_KEY", "test-key")
		provider, err := NewProvider(ProviderConfig{ProviderType: ProviderGemini})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("returns error for unknown provider", func(t *testing.T) {
		provider, err := NewProvider(ProviderConfig{ProviderType: ProviderUnknown})
		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider type 'unknown' is not supported")
	})

	t.Run("returns error for unsupported provider", func(t *testing.T) {
		provider, err := NewProvider(ProviderConfig{ProviderType: "something-else"})
		assert.Nil(t, provider)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported provider: something-else")
	})
}
