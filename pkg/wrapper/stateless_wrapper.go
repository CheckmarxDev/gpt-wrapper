package wrapper

import (
	"errors"

	"github.com/checkmarxDev/gpt-wrapper/internal"
	"github.com/checkmarxDev/gpt-wrapper/internal/secrets"
	"github.com/checkmarxDev/gpt-wrapper/pkg/maskedSecret"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/models"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
)

const OpenAiEndPoint = "https://api.openai.com/v1/chat/completions"

type StatelessWrapper interface {
	Call([]message.Message, []message.Message) ([]message.Message, error)
	SetupCall([]message.Message)
	MaskSecrets(fileContent string) (*maskedSecret.MaskedEntry, error)
}

type StatelessWrapperImpl struct {
	wrapper internal.Wrapper
	model   string
	dropLen int
	limit   int
}

func NewStatelessWrapper(endPoint, apiKey, model string, dropLen, limit int) (StatelessWrapper, error) {
	if model == "" {
		model = models.DefaultModel
	}
	wrapper, err := internal.NewWrapperFactory(endPoint, apiKey, dropLen)
	if err != nil {
		return nil, err
	}
	return &StatelessWrapperImpl{
		wrapper,
		model,
		dropLen,
		limit,
	}, nil
}

func (w *StatelessWrapperImpl) SetupCall(setupMessages []message.Message) {
	w.wrapper.SetupCall(setupMessages)
}

func (w *StatelessWrapperImpl) Call(history []message.Message, newMessages []message.Message) ([]message.Message, error) {
	var conversation []message.Message
	userMessageCount := 0
	for _, m := range append(history, newMessages...) {
		maskedContent, _, err := secrets.MaskSecrets(m.Content)
		if err != nil {
			return nil, err
		}
		conversation = append(conversation, message.Message{Role: m.Role, Content: maskedContent})
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

	response, err := w.wrapper.Call(requestBody)
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

func (w *StatelessWrapperImpl) MaskSecrets(fileContent string) (*maskedSecret.MaskedEntry, error) {
	maskedFile, maskedSecrets, err := secrets.MaskSecrets(fileContent)
	if err != nil {
		return nil, err
	}
	var maskedEntry = maskedSecret.MaskedEntry{}
	maskedEntry.MaskedFile = maskedFile
	maskedEntry.MaskedSecrets = maskedSecrets
	return &maskedEntry, nil
}
