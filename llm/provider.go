package llm

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

func NewProvider(config ProviderConfig) (Provider, error) {
	log.Debug().Msgf("providerConfig %+v", config)

	switch config.ProviderType {
	case ProviderOllama:
		return NewOllamaProvider(config), nil
	case ProviderLmStudio:
		return NewLmStudioProvider(config), nil
	case ProviderUnknown:
		return nil, fmt.Errorf("llm: provider type 'unknown' is not supported")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.ProviderType)
	}
}
