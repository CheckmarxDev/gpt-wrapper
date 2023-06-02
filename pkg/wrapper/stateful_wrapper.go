package wrapper

import (
	"github.com/checkmarxdev/gpt-wrapper/pkg/connector"
	"github.com/checkmarxdev/gpt-wrapper/pkg/message"

	"github.com/google/uuid"
)

type StatefulWrapper interface {
	GenerateId() uuid.UUID
	Call(uuid.UUID, []message.Message) ([]message.Message, error)
}

type StatefulWrapperImpl struct {
	connector connector.Connector
	StatelessWrapper
}

func NewStatefulWrapper(storageConnector connector.Connector, apikey, model string) StatefulWrapper {
	return StatefulWrapperImpl{
		storageConnector,
		NewStatelessWrapper(apikey, model),
	}
}

func (w StatefulWrapperImpl) GenerateId() uuid.UUID {
	return uuid.New()
}

func (w StatefulWrapperImpl) Call(id uuid.UUID, newMessages []message.Message) ([]message.Message, error) {
	var err error
	var history []message.Message
	var response []message.Message

	history, err = w.connector.HistoryById(id)
	if err != nil {
		return nil, err
	}

	response, err = w.StatelessWrapper.Call(history, newMessages)
	if err != nil {
		return nil, err
	}
	if len(response) != 1 {
		panic(response)
	}

	history = append(history, newMessages...)
	history = append(history, response[0])

	err = w.connector.SaveHistory(id, history)
	if err != nil {
		return nil, err
	}

	return response, nil
}
