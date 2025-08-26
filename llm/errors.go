package llm

import "github.com/sashabaranov/go-openai"

type OpenAIError struct {
	*openai.RequestError
}

func (e *OpenAIError) HTTPStatusCode() int {
	return e.RequestError.HTTPStatusCode
}

func wrapOpenAIError(err error) error {
	if reqErr, ok := err.(*openai.RequestError); ok {
		return &OpenAIError{RequestError: reqErr}
	}
	return err
}
