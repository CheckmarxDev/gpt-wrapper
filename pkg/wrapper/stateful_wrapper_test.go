package wrapper

import (
	"fmt"
	"github.com/spf13/viper"
	"testing"

	"github.com/checkmarxDev/gpt-wrapper/pkg/connector"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/models"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
)

type Config struct {
	EndPointGRPC string `mapstructure:"ENDPOINT_GRPC"`
	EndPointRest string `mapstructure:"ENDPOINT_REST"`
	ApiKey       string `mapstructure:"GPT_APIKEY"`
}

func TestCallGPT_FS(t *testing.T) {
	var history []message.Message
	wrapper, err := NewStatefulWrapper(connector.NewFileSystemConnector(""), "https://api.openai.com/v1/chat/completions", apikey, models.GPT3Dot5Turbo, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	id := wrapper.GenerateId()
	t.Log(id)
	for _, q := range userQuestions {
		t.Log(q)
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

func TestCallGPT_ToProxy(t *testing.T) {
	cfg := Config{}
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	if err := viper.ReadInConfig(); err != nil {
		t.Fatal("Error reading env file", err)
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	var history []message.Message
	wrapper, err := NewStatefulWrapper(connector.NewFileSystemConnector(""), cfg.EndPointGRPC, apikey, models.GPT4, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	id := wrapper.GenerateId()
	t.Log(id)
	for _, q := range userQuestions {
		t.Log(q)
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

func TestMaskSecrets(t *testing.T) {
	wrapper, err := NewStatefulWrapper(connector.NewFileSystemConnector(""), "https://api.openai.com/v1/chat/completions", apikey, models.GPT3Dot5Turbo, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	id := wrapper.GenerateId()
	t.Log(id)
	entries, err := wrapper.MaskSecrets("password=exposed")
	if err != nil {
		return
	}
	if len(entries.MaskedSecrets) > 0 {
		t.Logf("secret: %s, masked: %s, line: %d\n", entries.MaskedSecrets[0].Secret, entries.MaskedSecrets[0].Masked, entries.MaskedSecrets[0].Line)
	}
}
