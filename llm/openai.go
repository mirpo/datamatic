package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

const DefaultOpenAIBaseURL = "https://api.openai.com/v1"

type OpenAIProvider struct {
	config ProviderConfig
	client *openai.Client
}

func NewOpenAIProvider(config ProviderConfig) *OpenAIProvider {
	clientConfig := openai.DefaultConfig(config.AuthToken)

	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	return &OpenAIProvider{
		config: config,
		client: openai.NewClientWithConfig(clientConfig),
	}
}

type ResponseJSONSchema struct {
	Name   string      `json:"name"`
	Strict bool        `json:"strict"`
	Schema interface{} `json:"schema"`
}

type ResponseFormat struct {
	Type       string             `json:"type,omitempty"`
	JSONSchema ResponseJSONSchema `json:"json_schema,omitempty"`
}

func (p *OpenAIProvider) Generate(ctx context.Context, request GenerateRequest) (*GenerateResponse, error) {
	req := openai.ChatCompletionRequest{
		Model:  p.config.ModelName,
		Stream: false,
	}

	if p.config.Temperature != nil {
		req.Temperature = float32(*p.config.Temperature)
	}
	if p.config.MaxTokens != nil {
		req.MaxTokens = *p.config.MaxTokens
	}

	messages := []openai.ChatCompletionMessage{}

	if request.SystemMessage != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: request.SystemMessage,
		})
	}

	if len(request.Base64Image) == 0 {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: request.UserMessage,
		})
	} else {
		imageURL := fmt.Sprintf("data:image/jpeg;base64,%s", request.Base64Image)
		messages = append(messages, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleUser,
			MultiContent: []openai.ChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: request.UserMessage,
				},
				{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL: imageURL,
					},
				},
			},
		})
	}

	req.Messages = messages

	if request.IsJSON && request.JSONSchema != nil {
		rawSchema, err := json.Marshal(request.JSONSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema: %w", err)
		}

		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "json_schema",
				Strict: true,
				Schema: json.RawMessage(rawSchema),
			},
		}
	}

	log.Debug().Msgf("LLM request: model=%s, messages=%d, to baseUrl: %s", req.Model, len(req.Messages), p.config.BaseURL)

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm: openai: completion request failed: %w", err)
	}

	if resp.Model != p.config.ModelName {
		log.Warn().Msgf("OpenAI response: model mismatch: expected %s, got %s", p.config.ModelName, resp.Model)
	}

	log.Debug().Msgf("OpenAI response: model=%s, choices=%d, usage=%+v", resp.Model, len(resp.Choices), resp.Usage)

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("llm: openai: received no choices in response")
	}

	return &GenerateResponse{
		Text: resp.Choices[0].Message.Content,
	}, nil
}
