package connector

import (
	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"

	"github.com/google/uuid"
)

type Connector interface {
	HistoryById(uuid.UUID) ([]message.Message, error)
	DeleteHistory(uuid.UUID) error
	SaveHistory(uuid.UUID, []message.Message) error
}
