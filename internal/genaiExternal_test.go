package internal

import (
	"bytes"
	"encoding/json"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestHandleGptResponse(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte("test"))
	}))
	defer testServer.Close()
	wrapper := WrapperImpl{
		apiKey:   "test",
		endPoint: testServer.URL,
		dropLen:  4}
	resp := &http.Response{}
	_, err := wrapper.handleGptResponse("test", &message.MetaData{}, &ChatCompletionRequest{}, resp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandleGptResponseNegativeOpenAi(t *testing.T) {
	wrapper := WrapperImpl{
		apiKey:   "test",
		endPoint: "testServer.URL",
		dropLen:  4}
	gptError := GptError{
		Message: "test",
		Type:    "test",
		Param:   "test",
		Code:    429,
	}
	errRes := ErrorResponse{
		Error: gptError,
	}
	errResB, _ := json.Marshal(errRes)
	resp := &http.Response{Body: io.NopCloser(bytes.NewReader(errResB))}
	resp.StatusCode = http.StatusTooManyRequests
	res, err := wrapper.handleGptResponse("test", nil, &ChatCompletionRequest{}, resp)
	if res != nil {
		t.Fatal("Expected nil response")
	}
	if err == nil {
		t.Fatal("Expected error")
	}
	assert.Contains(t, err.Error(), "test")
	assert.Contains(t, err.Error(), strconv.Itoa(http.StatusTooManyRequests))
}

func TestHandleGptResponseNegativeAzureExternal(t *testing.T) {
	wrapper := WrapperImpl{
		apiKey:   "test",
		endPoint: "testServer.URL",
		dropLen:  4}
	gptError := GptError{
		Message: "test",
		Type:    "test",
		Param:   "test",
		Code:    429,
	}
	errRes := ErrorResponse{
		Error: gptError,
	}
	errResB, _ := json.Marshal(errRes)
	resp := &http.Response{Body: io.NopCloser(bytes.NewReader(errResB))}
	resp.Header = make(http.Header)
	resp.Header.Set("X-Gen-Ai-ErrorCode", strconv.Itoa(http.StatusTooManyRequests))
	resp.StatusCode = http.StatusFailedDependency
	res, err := wrapper.handleGptResponse("test", &message.MetaData{
		RequestID: "test",
		TenantID:  "test",
		UserAgent: "test",
		Feature:   "test",
	}, &ChatCompletionRequest{}, resp)
	if res != nil {
		t.Fatal("Expected nil response")
	}
	if err == nil {
		t.Fatal("Expected error")
	}
	assert.Contains(t, err.Error(), "test")
	assert.Contains(t, err.Error(), strconv.Itoa(http.StatusTooManyRequests))
	assert.NotContains(t, err.Error(), strconv.Itoa(http.StatusFailedDependency))
}

func TestHandleGptResponseNegativeAzureInternal(t *testing.T) {
	wrapper := WrapperImpl{
		apiKey:   "test",
		endPoint: "testServer.URL",
		dropLen:  4}
	gptError := GptError{
		Message: "test",
		Type:    "test",
		Param:   "test",
		Code:    429,
	}
	errRes := ErrorResponse{
		Error: gptError,
	}
	errResB, _ := json.Marshal(errRes)
	resp := &http.Response{Body: io.NopCloser(bytes.NewReader(errResB))}
	resp.Header = make(http.Header)
	resp.Header.Set("X-Gen-Ai-ErrorCode", strconv.Itoa(0))
	resp.StatusCode = http.StatusInternalServerError
	res, err := wrapper.handleGptResponse("test", &message.MetaData{
		RequestID: "test",
		TenantID:  "test",
		UserAgent: "test",
		Feature:   "test",
	}, &ChatCompletionRequest{}, resp)
	if res != nil {
		t.Fatal("Expected nil response")
	}
	if err == nil {
		t.Fatal("Expected error")
	}
	assert.NotContains(t, err.Error(), "test")
	assert.Contains(t, err.Error(), strconv.Itoa(0))
	assert.Contains(t, err.Error(), strconv.Itoa(http.StatusInternalServerError))
}
