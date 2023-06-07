package connector

import (
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"

	"github.com/google/uuid"
)

type Connector interface {
	HistoryById(uuid.UUID) ([]message.Message, error)
	SaveHistory(uuid.UUID, []message.Message) error
}
