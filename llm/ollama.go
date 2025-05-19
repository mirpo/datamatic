package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/mirpo/datamatic/httpclient"
	"github.com/rs/zerolog/log"
)

const DefaultOllamaBaseURL = "http://localhost:11434"

type OllamaProvider struct {
	config ProviderConfig
	client *httpclient.Client
}

func NewOllamaProvider(config ProviderConfig) *OllamaProvider {
	if config.BaseURL == "" {
		config.BaseURL = DefaultOllamaBaseURL
	}

	return &OllamaProvider{
		config: config,
		client: httpclient.NewClient(config.BaseURL, config.AuthToken, httpclient.WithTimeout(time.Duration(config.HTTPTimeout)*time.Second)),
	}
}

type ollamaChatRequest struct {
	Model    string           `json:"model"`
	Stream   bool             `json:"stream"`
	Messages []ollamaMessage  `json:"messages"`
	Options  ollamaReqOptions `json:"options,omitempty"`
	Format   interface{}      `json:"format,omitempty"`
}

type ollamaReqOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumCtx      *int     `json:"num_ctx,omitempty"`
}

type ollamaMessage struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Images  *[]string `json:"images,omitempty"`
}

type ollamaChatResponse struct {
	Model   string        `json:"model"`
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

func (p *OllamaProvider) Generate(ctx context.Context, request GenerateRequest) (*GenerateResponse, error) {
	req := ollamaChatRequest{
		Model:  p.config.ModelName,
		Stream: false,
		Options: ollamaReqOptions{
			Temperature: p.config.Temperature,
			NumCtx:      p.config.MaxTokens,
		},
	}

	msgs := []ollamaMessage{}

	if request.SystemMessage != "" {
		msgs = append(msgs, ollamaMessage{Role: "system", Content: request.SystemMessage})
	}

	userMsg := ollamaMessage{Role: "user", Content: request.UserMessage}
	if len(request.Base64Image) > 0 {
		images := []string{
			request.Base64Image,
		}
		userMsg.Images = &images
	}
	msgs = append(msgs, userMsg)
	req.Messages = msgs

	if request.IsJSON {
		req.Format = &request.JSONSchema
	}

	log.Debug().Msgf("LLM request: %+v, to baseUrl: %s", req, p.client.BaseURL)

	var llmResponse ollamaChatResponse
	err := p.client.Post(ctx, "/api/chat", req, &llmResponse, nil)
	if err != nil {
		return nil, fmt.Errorf("llm: ollama: completion request failed: %w", err)
	}

	log.Debug().Msgf("Ollama response: model=%s, contents: '%+v'", llmResponse.Model, llmResponse)

	return &GenerateResponse{
		Text: llmResponse.Message.Content,
	}, nil
}
