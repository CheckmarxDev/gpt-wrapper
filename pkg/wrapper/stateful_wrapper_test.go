package wrapper

import (
	"fmt"
	"testing"

	"github.com/checkmarxDev/gpt-wrapper/pkg/connector"
	"github.com/checkmarxDev/gpt-wrapper/pkg/message"
	"github.com/checkmarxDev/gpt-wrapper/pkg/models"
	"github.com/checkmarxDev/gpt-wrapper/pkg/role"
)

func TestCallGPT_FS(t *testing.T) {
	var history []message.Message
	wrapper := NewStatefulWrapper(connector.NewFileSystemConnector(""), apikey, models.GPT3Dot5Turbo, 4, 0)
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
	wrapper := NewStatefulWrapper(connector.NewFileSystemConnector(""), apikey, models.GPT3Dot5Turbo, 4, 0)
	id := wrapper.GenerateId()
	t.Log(id)
	_, secrets, err := wrapper.MaskSecrets("resource \"serverscom_dedicated_server\" \"node_1\" {\n  password             = \"long password with spaces\"\n  location             = \"SJC1\"\n  server_model         = \"Dell R440 / 2xIntel Xeon Silver-4114 / 32 GB RAM / 1x480 GB SSD\"\n  ram_size             = 32\n\n  operating_system     = \"Ubuntu 20.04-server x86_64\"\n\n  private_uplink       = \"Private 10 Gbps with redundancy\"\n  public_uplink        = \"Public 10 Gbps with redundancy\"\n  bandwidth            = \"200000 GB\"\n  # ...\n  # Some parameters are not displayed here to shorten the specification.\n  # You can see the complete example of the resource in the relevant section of the documentation.\n  # ...\n user_data = <")
	if err != nil {
		return
	}
	if len(secrets) >0 {
		fmt.Printf("secret: %s, masked : %s \n", secrets[0].Secret,secrets[0].Masked)
	}
}

