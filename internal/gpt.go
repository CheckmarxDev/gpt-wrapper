package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const RateLimitWaitSeconds = 60

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []ChatCompletionMessage `json:"messages"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id,omitempty"`
	Choices []struct {
		Index        int                   `json:"index,omitempty"`
		Message      ChatCompletionMessage `json:"message"`
		FinishReason string                `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
}

type Wrapper interface {
	Call(request ChatCompletionRequest) (*ChatCompletionResponse, error)
}

type WrapperImpl struct {
	Apikey string
}

func NewWrapperImpl(apikey string) WrapperImpl {
	return WrapperImpl{
		Apikey: apikey,
	}
}

func (w WrapperImpl) Call(requestBody ChatCompletionRequest) (*ChatCompletionResponse, error) {
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.Apikey))

	return req, nil
}

func (w WrapperImpl) handleGptResponse(requestBody ChatCompletionRequest, resp *http.Response) (*ChatCompletionResponse, error) {
	var err error
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var responseBody = new(ChatCompletionResponse)
		err = json.Unmarshal(bodyBytes, responseBody)
		if err != nil {
			return nil, err
		}
		return responseBody, nil
	case http.StatusBadRequest:
		log.Println("model's maximum context length, trimming oldest 3 messages from context")
		return w.Call(ChatCompletionRequest{
			Model:    requestBody.Model,
			Messages: requestBody.Messages[3:],
		})
	case http.StatusUnauthorized:
		log.Println("api key issue")
		return nil, errors.New(string(bodyBytes))
	case http.StatusNotFound:
		log.Println("model not found")
		return nil, errors.New(string(bodyBytes))
	case http.StatusTooManyRequests:
		log.Printf("Too many requests, sleeping %d seconds\n", RateLimitWaitSeconds)
		time.Sleep(RateLimitWaitSeconds * time.Second)
		return w.Call(requestBody)
	case http.StatusInternalServerError:
		log.Println("gpt error")
		return nil, errors.New(string(bodyBytes))
	default:
		log.Printf("response code %d\n", resp.StatusCode)
		return nil, errors.New(string(bodyBytes))
	}
}
