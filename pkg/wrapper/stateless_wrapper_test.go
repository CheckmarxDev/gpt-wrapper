package wrapper

import (
	"fmt"
	"testing"

	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/model"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
)

func TestCallGPT(t *testing.T) {
	var history []message.Message
	var response []message.Message
	wrapper := NewStatelessWrapper(apikey, model.Model)
	for _, q := range userQuestions {
		t.Log(q)
		var err error
		var newMessages []message.Message
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

		response, err = wrapper.Call(history, newMessages)
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
