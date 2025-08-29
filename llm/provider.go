package llm

import (
	"fmt"
	"os"

	"github.com/mirpo/datamatic/defaults"
	"github.com/rs/zerolog/log"
)

func NewProvider(config ProviderConfig) (Provider, error) {
	log.Debug().Msgf("llm: providerConfig %+v", config)

	switch config.ProviderType {
	case ProviderOllama:
		if config.BaseURL == "" {
			config.BaseURL = defaults.OllamaURL
		}
		return NewOpenAIProvider(config), nil
	case ProviderLmStudio:
		if config.BaseURL == "" {
			config.BaseURL = defaults.LMStudioURL
		}
		return NewOpenAIProvider(config), nil
	case ProviderOpenAI:
		token := os.Getenv("OPENAI_API_KEY")
		if token == "" {
			return nil, fmt.Errorf("llm: OPENAI_API_KEY environment variable is not set")
		}
		config.AuthToken = token
		return NewOpenAIProvider(config), nil
	case ProviderOpenRouter:
		token := os.Getenv("OPENROUTER_API_KEY")
		if token == "" {
			return nil, fmt.Errorf("llm: OPENROUTER_API_KEY environment variable is not set")
		}
		config.AuthToken = token
		if config.BaseURL == "" {
			config.BaseURL = defaults.OpenRouterURL
		}
		return NewOpenAIProvider(config), nil
	case ProviderGemini:
		token := os.Getenv("GEMINI_API_KEY")
		if token == "" {
			return nil, fmt.Errorf("llm: GEMINI_API_KEY environment variable is not set")
		}
		config.AuthToken = token
		if config.BaseURL == "" {
			config.BaseURL = defaults.GeminiURL
		}
		return NewOpenAIProvider(config), nil
	case ProviderUnknown:
		return nil, fmt.Errorf("llm: provider type 'unknown' is not supported")
	default:
		return nil, fmt.Errorf("llm: unsupported provider: %s", config.ProviderType)
	}
}
