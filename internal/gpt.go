package internal

import (
	"errors"
	"fmt"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/role"
	"net/url"
)

// const gptByAzure = "https://cxgpt4.openai.azure.com/openai/deployments/gpt-4/chat/completions?api-version=2023-05-15"
// const gptByOpenAi = "https://api.openai.com/v1/chat/completions"

type ChatMetaData struct {
	TenantID  string `json:"tenant_id"`
	RequestID string `json:"request_id"`
	Origin    string `json:"origin"`
}

type ChatCompletionRequest struct {
	Model    string            `json:"model"`
	Messages []message.Message `json:"messages"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id,omitempty"`
	Choices []struct {
		Index        int             `json:"index,omitempty"`
		Message      message.Message `json:"message"`
		FinishReason string          `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
	Usage struct {
		TotalTokens      int `json:"total_tokens,omitempty"`
		CompletionTokens int `json:"completion_tokens,omitempty"`
		PromptTokens     int `json:"prompt_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

type ErrorResponse struct {
	Error struct {
		Message string      `json:"message,omitempty"`
		Type    string      `json:"type,omitempty"`
		Param   string      `json:"param,omitempty"`
		Code    interface{} `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

type Wrapper interface {
	Call(request ChatCompletionRequest) (*ChatCompletionResponse, error)
	SetupCall(messages []message.Message)
	Close() error
}

func NewWrapperFactory(endPoint, apiKey string, dropLen int) (Wrapper, error) {
	endPointURL, err := url.Parse(endPoint)
	if err != nil {
		return nil, err
	}
	if endPointURL.Scheme == "http" || endPointURL.Scheme == "https" {
		return NewWrapperImpl(endPoint, apiKey, dropLen), nil
	}
	return NewWrapperInternalImpl(endPoint, dropLen)
}

func fromResponse(statusCode int, e *ErrorResponse) error {
	var msg string
	if e.Error.Message != "" {
		msg = e.Error.Message
	} else {
		msg = fmt.Sprintf("%v", e.Error.Code)
	}

	msg = fmt.Sprintf("Error Code: %d, %s", statusCode, msg)

	return errors.New(msg)
}

func findLastUserIndex(messages []message.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == role.User {
			return i
		}
	}
	return 0
}
