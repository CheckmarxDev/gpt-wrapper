package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Checkmarx/gen-ai-wrapper/internal/api/redirect_prompt"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WrapperInternalImpl struct {
	connection    *grpc.ClientConn
	client        redirect_prompt.AiProxyServiceClient
	dropLen       int
	setupMessages []message.Message
}

func NewWrapperInternalImpl(endPoint string, dropLen int) (Wrapper, error) {
	connection, err := grpc.NewClient(endPoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := redirect_prompt.NewAiProxyServiceClient(connection)
	return &WrapperInternalImpl{
		connection: connection,
		client:     client,
		dropLen:    dropLen,
	}, nil
}

func (w *WrapperInternalImpl) SetupCall(messages []message.Message) {
	w.setupMessages = messages
}

func (w *WrapperInternalImpl) Call(requestBody ChatCompletionRequest) (*ChatCompletionResponse, error) {
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

	req, err := w.prepareRequest(ChatMetaData{}, requestBody)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.RedirectPrompt(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return w.handleGptResponse(requestBody, resp)
}

func (w *WrapperInternalImpl) prepareRequest(metaData ChatMetaData, requestBody ChatCompletionRequest) (*redirect_prompt.RedirectPromptRequest, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req := &redirect_prompt.RedirectPromptRequest{
		Tenant:    metaData.TenantID,
		RequestId: metaData.RequestID,
		Origin:    metaData.Origin,
		Content:   jsonData,
	}
	return req, nil
}

func (w *WrapperInternalImpl) handleGptResponse(requestBody ChatCompletionRequest, resp *redirect_prompt.RedirectPromptResponse) (*ChatCompletionResponse, error) {
	var err error
	bodyBytes, err := io.ReadAll(bytes.NewBuffer(resp.Content))
	if err != nil {
		return nil, err
	}
	if resp.GenAiErrorCode == http.StatusOK {
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
	switch resp.GenAiErrorCode {
	case http.StatusBadRequest:
		if errorResponse.Error.Code == errorCodeMaxTokens {
			return w.Call(ChatCompletionRequest{
				Model:    requestBody.Model,
				Messages: requestBody.Messages[w.dropLen:],
			})
		}
	}
	return nil, fromResponse(int(resp.GenAiErrorCode), errorResponse)
}

func (w *WrapperInternalImpl) Close() error {
	return w.connection.Close()
}
