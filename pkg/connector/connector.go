package connector

import (
	"gpt-wrapper/pkg/message"

	"github.com/google/uuid"
)

type Connector interface {
	HistoryById(uuid.UUID) ([]message.Message, error)
	SaveHistory(uuid.UUID, []message.Message) error
}
