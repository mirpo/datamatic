package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/mirpo/datamatic/httpclient"
	"github.com/rs/zerolog/log"
)

const DefaultLmStudioBaseURL = "http://localhost:1234/v1"

type LmStudioProvider struct {
	config ProviderConfig
	client *httpclient.Client
}

func NewLmStudioProvider(config ProviderConfig) *LmStudioProvider {
	if config.BaseURL == "" {
		config.BaseURL = DefaultLmStudioBaseURL
	}

	return &LmStudioProvider{
		config: config,
		client: httpclient.NewClient(config.BaseURL, config.AuthToken, httpclient.WithTimeout(time.Duration(config.HTTPTimeout)*time.Second)),
	}
}

type lmStudioMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type lmStudioContentPart struct {
	Type     string           `json:"type"`
	Text     string           `json:"text,omitempty"`
	ImageURL lmStudioImageURL `json:"image_url"`
}

type lmStudioImageURL struct {
	URL string `json:"url"`
}

type lmStudioChatRequest struct {
	Model          string            `json:"model"`
	Messages       []lmStudioMessage `json:"messages"`
	Temperature    *float64          `json:"temperature,omitempty"`
	MaxTokens      *int              `json:"max_tokens,omitempty"`
	Stream         bool              `json:"stream"`
	ResponseFormat *ResponseFormat   `json:"response_format,omitempty"`
}

type lmStudioChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int             `json:"index"`
		Message      lmStudioMessage `json:"message"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *LmStudioProvider) Generate(ctx context.Context, request GenerateRequest) (*GenerateResponse, error) {
	req := lmStudioChatRequest{
		Model:       p.config.ModelName,
		Stream:      false,
		Temperature: p.config.Temperature,
		MaxTokens:   p.config.MaxTokens,
	}

	msgs := []lmStudioMessage{}
	if request.SystemMessage != "" {
		msgs = append(msgs, lmStudioMessage{Role: "system", Content: request.SystemMessage})
	}

	userMessage := lmStudioMessage{Role: "user"}

	// text request
	if len(request.Base64Image) == 0 {
		userMessage.Content = request.UserMessage
	} else {
		contentParts := []lmStudioContentPart{}

		// first text part
		contentParts = append(contentParts, lmStudioContentPart{
			Type: "text",
			Text: request.UserMessage,
		})

		// image part
		imageURL := fmt.Sprintf("data:image/jpeg;base64,%s", request.Base64Image)
		contentParts = append(contentParts, lmStudioContentPart{
			Type: "image_url",
			ImageURL: lmStudioImageURL{
				URL: imageURL,
			},
		})

		userMessage.Content = contentParts
	}

	msgs = append(msgs, userMessage)
	req.Messages = msgs

	if request.IsJSON {
		responseFormat := NewResponseFormat(request.JSONSchema)
		req.ResponseFormat = &responseFormat
	}

	log.Debug().Msgf("LLM request: %+v, to baseUrl: %s", req, p.client.BaseURL)

	var llmResponse lmStudioChatResponse
	err := p.client.Post(ctx, "/chat/completions", req, &llmResponse, nil)
	if err != nil {
		return nil, fmt.Errorf("llm: lmstudio: completion request failed: %w", err)
	}

	if llmResponse.Model != p.config.ModelName {
		log.Warn().Msgf("LM Studio response: model mismatch: expected %s, got %s", p.config.ModelName, llmResponse.Model)
	}

	log.Debug().Msgf("LM Studio response: model=%s, object=%+v", llmResponse.Model, llmResponse)

	if len(llmResponse.Choices) == 0 {
		return nil, fmt.Errorf("llm: lmstudio: received no choices in response")
	}

	responseContent, ok := llmResponse.Choices[0].Message.Content.(string)
	if !ok {
		return nil, fmt.Errorf("llm: lmstudio: response message content is not a string")
	}

	return &GenerateResponse{
		Text: responseContent,
	}, nil
}
