package connector

import (
	"encoding/json"
	"os"
	"path"

	"github.com/Checkmarx/gen-ai-wrapper/pkg/message"

	"github.com/google/uuid"
)

const innerDir = "cx-gpt"

type FileSystemConnector struct {
	BaseDir string
}

func NewFileSystemConnector(baseDir string) Connector {
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	return FileSystemConnector{
		BaseDir: baseDir,
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

func (w FileSystemConnector) DeleteHistory(id uuid.UUID) error {
	filePath := w.getFilePathById(id)

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return os.Remove(filePath)
}

func (w FileSystemConnector) SaveHistory(id uuid.UUID, history []message.Message) error {
	var err error
	filePath := w.getFilePathById(id)

	bytes, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return w.writeHistory(filePath, bytes)
}

func (w FileSystemConnector) writeHistory(filepath string, bytes []byte) error {
	var err error

	_, err = os.Stat(w.getBasePath())
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(w.getBasePath(), 0700)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return os.WriteFile(filepath, bytes, 0644)
}

func (w FileSystemConnector) getFilePathById(id uuid.UUID) string {
	return path.Join(w.getBasePath(), id.String())
}

func (w FileSystemConnector) getBasePath() string {
	return path.Join(w.BaseDir, innerDir)
}
