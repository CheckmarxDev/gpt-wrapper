package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/models"
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

func (w *WrapperImpl) Call(cxAuth string, metaData *message.MetaData, request *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if w.setupMessages != nil {
		//true for GPT4
		if request.Model == models.GPT4 {
			request.Messages = append(w.setupMessages, request.Messages...)
		} else {
			userIndex := findLastUserIndex(request.Messages)
			front := request.Messages[:userIndex]
			back := request.Messages[userIndex:]
			request.Messages = append(front, w.setupMessages...)
			request.Messages = append(request.Messages, back...)
		}
	}

	req, err := w.prepareRequest(cxAuth, metaData, request)
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

	return w.handleGptResponse(cxAuth, metaData, request, resp)
}

func (w *WrapperImpl) prepareRequest(cxAuth string, metaData *message.MetaData, requestBody *ChatCompletionRequest) (*http.Request, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, w.endPoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if metaData != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cxAuth))
		req.Header.Set("X-Request-ID", metaData.RequestID)
		req.Header.Set("X-Tenant-ID", metaData.TenantID)
		req.Header.Set("User-Agent", metaData.UserAgent)
		req.Header.Set("X-Feature", metaData.Feature)
		if metaData.ExternalModel != nil {
			req.Header.Set("X-External-Endpoint", metaData.ExternalModel.Endpoint)
			req.Header.Set("X-External-ApiKey", metaData.ExternalModel.ApiKey)
		}
	} else
	// headers suited for openAi endpoint
	{
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.apiKey))
	}
	return req, nil
}

func (w *WrapperImpl) handleGptResponse(accessToken string, metaData *message.MetaData, requestBody *ChatCompletionRequest, resp *http.Response) (*ChatCompletionResponse, error) {
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
			return w.Call(accessToken, metaData, &ChatCompletionRequest{
				Model:    requestBody.Model,
				Messages: requestBody.Messages[w.dropLen:],
			})
		}
	}
	return nil, fromResponse(resp.StatusCode, errorResponse)
}

func (w *WrapperImpl) Close() error {
	return nil
}
