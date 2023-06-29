package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/models"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
	"io"
	"net/http"
)

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
}

type WrapperImpl struct {
	apiKey        string
	dropLen       int
	setupMessages []message.Message
}

func NewWrapperImpl(apiKey string, dropLen int) WrapperImpl {
	return WrapperImpl{
		apiKey:  apiKey,
		dropLen: dropLen,
	}
}

func (w WrapperImpl) SetupCall(messages []message.Message) {
	w.setupMessages = messages
}

func (w WrapperImpl) Call(requestBody ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if w.setupMessages != nil {
		if requestBody.Model == models.GPT4 {
			requestBody.Messages = append(w.setupMessages, requestBody.Messages...)
		} else {
			userIndex := findLastUserIndex(requestBody.Messages)
			front := requestBody.Messages[:userIndex]
			back := requestBody.Messages[userIndex:]
			requestBody.Messages = append(front, w.setupMessages...)
			requestBody.Messages = append(requestBody.Messages, back...)
		}
	}

	req, err := w.prepareRequest(requestBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return w.handleGptResponse(requestBody, resp)
}

func (w WrapperImpl) prepareRequest(requestBody ChatCompletionRequest) (*http.Request, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.apiKey))

	return req, nil
}

func (w WrapperImpl) handleGptResponse(requestBody ChatCompletionRequest, resp *http.Response) (*ChatCompletionResponse, error) {
	var err error
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		var responseBody = new(ChatCompletionResponse)
		err = json.Unmarshal(bodyBytes, responseBody)
		if err != nil {
			return nil, err
		}
		return responseBody, nil
	}
	var errorResponse = new(ErrorResponse)
	err = json.Unmarshal(bodyBytes, errorResponse)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusBadRequest:
		if errorResponse.Error.Code == errorCodeMaxTokens {
			return w.Call(ChatCompletionRequest{
				Model:    requestBody.Model,
				Messages: requestBody.Messages[w.dropLen:],
			})
		}
	}
	return nil, fromResponse(errorResponse)
}

func fromResponse(e *ErrorResponse) error {
	var msg string
	if e.Error.Message != "" {
		msg = e.Error.Message
	} else {
		msg = fmt.Sprintf("%v", e.Error.Code)
	}
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
