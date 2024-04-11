package wrapper

import (
	"fmt"
	"testing"

	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/models"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
)

func TestCallGPT(t *testing.T) {
	var history []message.Message
	var response []message.Message
	wrapper, err := NewStatelessWrapper("", "741d17e7b77e4204a62e712bb5304615", models.GPT4, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range userQuestions {
		t.Log(q)
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

func TestCallEmptyApiKey(t *testing.T) {
	wrapper, err := NewStatelessWrapper("", "", models.GPT3Dot5Turbo, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	q := userQuestions[0]
	_, err = wrapper.Call(nil, []message.Message{{
		Role:    role.User,
		Content: fmt.Sprintf(userInput, q),
	}})
	if err == nil {
		t.Fatal("Call succeeded without API key")
	}
}
