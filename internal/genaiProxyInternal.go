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

type WrapperImpl struct {
	apiKey        string
	endPoint      string
	dropLen       int
	setupMessages []message.Message
}

func NewWrapperImpl(endPoint, apiKey string, dropLen int) Wrapper {
	return &WrapperImpl{
		endPoint: endPoint,
		apiKey:   apiKey,
		dropLen:  dropLen,
	}
}

func (w *WrapperImpl) SetupCall(messages []message.Message) {
	w.setupMessages = messages
}

func (w *WrapperImpl) Call(requestBody ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if w.setupMessages != nil {
		//true for GPT4
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

func (w *WrapperImpl) prepareRequest(requestBody ChatCompletionRequest) (*http.Request, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, w.endPoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.apiKey))
	return req, nil
}

func (w *WrapperImpl) handleGptResponse(requestBody ChatCompletionRequest, resp *http.Response) (*ChatCompletionResponse, error) {
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
	return nil, fromResponse(resp.StatusCode, errorResponse)
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
