package pkg

import (
	"gpt-wrapper/internal"
	"gpt-wrapper/pkg/message"
)

type StatelessWrapper interface {
	Call([]message.Message, []message.Message) ([]message.Message, error)
}

type StatelessWrapperImpl struct {
	Apikey string
	Model  string
}

func NewStatelessWrapper(apikey, model string) StatelessWrapper {
	return StatelessWrapperImpl{
		Apikey: apikey,
		Model:  model,
	}
}

func (w StatelessWrapperImpl) Call(history []message.Message, newMessages []message.Message) ([]message.Message, error) {
	var conversation []internal.ChatCompletionMessage

	for _, m := range append(history, newMessages...) {
		conversation = append(conversation, internal.ChatCompletionMessage{Role: m.Role, Content: m.Content})
	}

	requestBody := internal.ChatCompletionRequest{
		Model:    w.Model,
		Messages: conversation,
	}

	wrapper := internal.NewWrapperImpl(w.Apikey)

	response, err := wrapper.Call(requestBody)
	if err != nil {
		return nil, err
	}

	var responseMessages []message.Message

	for _, c := range response.Choices {
		responseMessages = append(responseMessages, message.Message{
			Role:    c.Message.Role,
			Content: c.Message.Content,
		})
	}

	return responseMessages, nil
}
