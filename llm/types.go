package llm

import "context"

type ProviderType string

const (
	ProviderOllama   ProviderType = "ollama"
	ProviderLmStudio ProviderType = "lmstudio"
	ProviderUnknown  ProviderType = "unknown"
)

type ProviderConfig struct {
	ProviderType ProviderType
	BaseURL      string
	ModelName    string
	AuthToken    string
	Temperature  *float64
	MaxTokens    *int
	HTTPTimeout  int
}

type Provider interface {
	Generate(ctx context.Context, request GenerateRequest) (*GenerateResponse, error)
}

type GenerateRequest struct {
	UserMessage   string
	SystemMessage string
}

type GenerateResponse struct {
	Text string
}
