package wrapper

import (
	"errors"
	"github.com/checkmarxDev/gpt-wrapper/pkg/connector"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"

	"github.com/google/uuid"
)

type StatefulWrapper interface {
	GenerateId() uuid.UUID
	Call(uuid.UUID, []message.Message) ([]message.Message, error)
}

type StatefulWrapperImpl struct {
	connector connector.Connector
	limit     int
	StatelessWrapper
}

func NewStatefulWrapper(storageConnector connector.Connector, apiKey, model string, dropLen int, limit int) StatefulWrapper {
	return StatefulWrapperImpl{
		storageConnector,
		limit,
		NewStatelessWrapper(apiKey, model, dropLen),
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

	allMessages := append(history, newMessages...)

	userMessageCount := 0
	for _, msg := range allMessages {
		if msg.Role == role.User {
			userMessageCount++
		}
	}

	if w.limit > 0 && userMessageCount > w.limit {
		return nil, errors.New("user message limit exceeded")
	}

	response, err = w.StatelessWrapper.Call(history, newMessages)
	if err != nil {
		return nil, err
	}
	if len(response) != 1 {
		panic(response)
	}

	allMessages = append(allMessages, response[0])

	err = w.connector.SaveHistory(id, allMessages)
	if err != nil {
		return nil, err
	}

	return response, nil
}
