package llm

import (
	"context"

	"github.com/mirpo/datamatic/jsonl"
)

type ProviderType string

const (
	ProviderOllama     ProviderType = "ollama"
	ProviderLmStudio   ProviderType = "lmstudio"
	ProviderOpenAI     ProviderType = "openai"
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderGemini     ProviderType = "gemini"
	ProviderUnknown    ProviderType = "unknown"
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
	IsJSON        bool
	JSONSchema    jsonl.JSONSchema
	Base64Image   string
}

type GenerateResponse struct {
	Text string
}
