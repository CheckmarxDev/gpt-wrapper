package wrapper

import (
	"errors"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/connector"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/maskedSecret"
	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"
	"github.com/google/uuid"
)

type StatefulWrapper interface {
	GenerateId() uuid.UUID
	SecureCall(cxAuth string, metaData *message.MetaData, historyID uuid.UUID, newMessages []message.Message) ([]message.Message, error)
	Call(historyID uuid.UUID, newMessages []message.Message) ([]message.Message, error)
	SetupCall(setupMessages []message.Message)
	MaskSecrets(fileContent string) (*maskedSecret.MaskedEntry, error)
}

type StatefulWrapperImpl struct {
	connector connector.Connector
	StatelessWrapper
}

func NewStatefulWrapperNew(storageConnector connector.Connector, endpoint, apiKey, model string, dropLen, limit int) (StatefulWrapper, error) {
	statelessWrapper, err := NewStatelessWrapper(endpoint, apiKey, model, dropLen, limit)
	if err != nil {
		return nil, err
	}
	return &StatefulWrapperImpl{
		storageConnector,
		statelessWrapper,
	}, nil
}

// NewStatefulWrapper will be deprecated in the future
func NewStatefulWrapper(storageConnector connector.Connector, apiKey, model string, dropLen, limit int) StatefulWrapper {
	statelessWrapper, err := NewStatefulWrapperNew(storageConnector, OpenAiEndPoint, apiKey, model, dropLen, limit)
	if err != nil {
		return nil
	}
	return statelessWrapper
}

func (w *StatefulWrapperImpl) SetupCall(setupMessages []message.Message) {
	w.StatelessWrapper.SetupCall(setupMessages)
}

func (w *StatefulWrapperImpl) GenerateId() uuid.UUID {
	return uuid.New()
}

func (w *StatefulWrapperImpl) SecureCall(cxAuth string, metaData *message.MetaData, historyID uuid.UUID, newMessages []message.Message) ([]message.Message, error) {
	var err error
	var history []message.Message
	var response []message.Message

	history, err = w.connector.HistoryById(historyID)
	if err != nil {
		return nil, err
	}

	response, err = w.StatelessWrapper.SecureCall(cxAuth, metaData, history, newMessages)
	if err != nil {
		return nil, err
	}
	if len(response) != 1 {
		return nil, errors.New("unexpected response length")
	}

	history = append(history, newMessages...)
	history = append(history, response[0])

	err = w.connector.SaveHistory(historyID, history)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (w *StatefulWrapperImpl) Call(historyID uuid.UUID, newMessages []message.Message) ([]message.Message, error) {
	return w.SecureCall("", nil, historyID, newMessages)
}

func (w *StatefulWrapperImpl) MaskSecrets(fileContent string) (*maskedSecret.MaskedEntry, error) {
	maskedSecrets, err := w.StatelessWrapper.MaskSecrets(fileContent)
	if err != nil {
		return nil, err
	}
	return maskedSecrets, nil
}
