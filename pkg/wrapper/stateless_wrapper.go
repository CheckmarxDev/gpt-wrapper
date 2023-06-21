package wrapper

import (
	"errors"
	"github.com/checkmarxDev/gpt-wrapper/internal"
	"github.com/checkmarxDev/gpt-wrapper/internal/secrets"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/models"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
)

type StatelessWrapper interface {
	Call([]message.Message, []message.Message) ([]message.Message, error)
}

type StatelessWrapperImpl struct {
	apiKey  string
	model   string
	dropLen int
	limit   int
}

func NewStatelessWrapper(apiKey, model string, dropLen, limit int) StatelessWrapper {
	if model == "" {
		model = models.DefaultModel
	}
	return StatelessWrapperImpl{
		apiKey,
		model,
		dropLen,
		limit,
	}
}

func (w StatelessWrapperImpl) Call(history []message.Message, newMessages []message.Message) ([]message.Message, error) {
	var conversation []internal.ChatCompletionMessage
	userMessageCount := 0
	for _, m := range append(history, newMessages...) {
		maskedContent, err := secrets.MaskSecrets(m.Content)
		if err != nil {
			return nil, err
		}
		conversation = append(conversation, internal.ChatCompletionMessage{Role: m.Role, Content: maskedContent})
		if m.Role == role.User {
			userMessageCount++
		}
	}

	if w.limit > 0 && userMessageCount > w.limit {
		return nil, errors.New("user message limit exceeded")
	}

	requestBody := internal.ChatCompletionRequest{
		Model:    w.model,
		Messages: conversation,
	}

	wrapper := internal.NewWrapperImpl(w.apiKey, 1)

	response, err := wrapper.Call(requestBody)
	if err != nil {
		return nil, err
	}

	var responseMessages []message.Message

	for _, c := range response.Choices {
		if c.FinishReason == internal.FinishReasonLength {
			return w.Call(history[w.dropLen:], newMessages)
		}
		responseMessages = append(responseMessages, message.Message{
			Role:    c.Message.Role,
			Content: c.Message.Content,
		})
	}

	return responseMessages, nil
}
