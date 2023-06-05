package pkg

import (
	"fmt"
	"testing"

	"github.com/checkmarxdev/gpt-wrapper/internal/model"
	"github.com/checkmarxdev/gpt-wrapper/internal/role"
	"github.com/checkmarxdev/gpt-wrapper/pkg/connector"
	"github.com/checkmarxdev/gpt-wrapper/pkg/message"
)

func TestCallGPT_FS(t *testing.T) {
	var history []message.Message
	wrapper := NewStatefulWrapper(connector.NewFileSystemConnector(""), apikey, model.Model)
	id := wrapper.GenerateId()
	for _, q := range userQuestions {
		var err error
		var newMessages []message.Message
		var response []message.Message
		newMessages = append(newMessages, message.Message{
			Role:    role.System,
			Content: systemInput,
		})
		newMessages = append(newMessages, message.Message{
			Role:    role.Assistant,
			Content: assistantInput,
		})
		newMessages = append(newMessages, message.Message{
			Role:    role.User,
			Content: fmt.Sprintf(userInput, q),
		})

		response, err = wrapper.Call(id, newMessages)
		if err != nil {
			t.Fatal(err)
		}
		if len(response) != 1 {
			t.Fatalf("Got multiple choices\n%v\n", response)
		}

		history = append(history, newMessages...)
		history = append(history, response[0])
	}
	for _, m := range history {
		t.Logf("%s\n\n%s\n\n", m.Role, m.Content)
	}
}
