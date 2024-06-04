package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func (w *WrapperInternalImpl) Call(_ string, metaData *message.MetaData, request *ChatCompletionRequest) (*ChatCompletionResponse, error) {
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

	req, err := w.prepareRequest(metaData, request)
	if err != nil {
		return nil, err
	}

	resp, err := w.client.RedirectPrompt(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return w.handleGptResponse(metaData, request, resp)
}

func (w *WrapperInternalImpl) prepareRequest(metaData *message.MetaData, requestBody *ChatCompletionRequest) (*redirect_prompt.RedirectPromptRequest, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	if metaData == nil {
		return nil, errors.New("metadata is nil")
	}
	req := &redirect_prompt.RedirectPromptRequest{
		Content:   jsonData,
		Tenant:    metaData.TenantID,
		RequestId: metaData.RequestID,
		Origin:    metaData.UserAgent,
		Feature:   metaData.Feature,
	}
	return req, nil
}

func (w *WrapperInternalImpl) handleGptResponse(metaData *message.MetaData, requestBody *ChatCompletionRequest, resp *redirect_prompt.RedirectPromptResponse) (*ChatCompletionResponse, error) {
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
			return w.Call("", metaData, &ChatCompletionRequest{
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
