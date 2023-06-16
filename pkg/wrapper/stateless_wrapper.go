package wrapper

import (
	"github.com/checkmarxDev/gpt-wrapper/internal"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/models"
)

type StatelessWrapper interface {
	Call([]message.Message, []message.Message) ([]message.Message, error)
}

type StatelessWrapperImpl struct {
	apiKey  string
	model   string
	dropLen int
}

func NewStatelessWrapper(apiKey, model string, dropLen int) StatelessWrapper {
	if model == "" {
		model = models.DefaultModel
	}
	return StatelessWrapperImpl{
		apiKey:  apiKey,
		model:   model,
		dropLen: dropLen,
	}
}

func (w StatelessWrapperImpl) Call(history []message.Message, newMessages []message.Message) ([]message.Message, error) {
	var conversation []internal.ChatCompletionMessage

	for _, m := range append(history, newMessages...) {
		conversation = append(conversation, internal.ChatCompletionMessage{Role: m.Role, Content: m.Content})
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
