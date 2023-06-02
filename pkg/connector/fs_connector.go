package connector

import (
	"encoding/json"
	"gpt-wrapper/pkg/message"
	"os"
	"path"

	"github.com/google/uuid"
)

const innerDir = "cx-gpt"

type FileSystemConnector struct {
	BaseDir string
}

func NewFileSystemConnector() Connector {
	return FileSystemConnector{
		BaseDir: os.TempDir(),
	}
}

func (w FileSystemConnector) HistoryById(id uuid.UUID) ([]message.Message, error) {
	var err error
	filePath := w.getFilePathById(id)

	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var history []message.Message
	err = json.Unmarshal(bytes, &history)
	if err != nil {
		return nil, err
	}

	return history, nil
}

func (w FileSystemConnector) SaveHistory(id uuid.UUID, history []message.Message) error {
	var err error
	filePath := w.getFilePathById(id)

	bytes, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, bytes, 0644)
}

func (w FileSystemConnector) getFilePathById(id uuid.UUID) string {
	return path.Join(w.BaseDir, innerDir, id.String())
}
