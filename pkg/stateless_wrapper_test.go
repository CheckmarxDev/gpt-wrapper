package pkg

import (
	"fmt"
	"os"
	"testing"

	"github.com/checkmarxdev/gpt-wrapper/internal/model"
	"github.com/checkmarxdev/gpt-wrapper/internal/role"
	"github.com/checkmarxdev/gpt-wrapper/pkg/message"
)

var apikey = os.Getenv("GPT-APIKEY")

const systemInput = `You are the Checkmarx KICS bot who can answer technical questions related to the results of KICS.

You should be able to analyze and understand both the technical aspects of the security results and the common queries users may have about the results.

You should also be capable of delivering clear, concise, and informative answers to help take appropriate action based on the findings.

 

If a question irrelevant to the mentioned KICS source or result is asked, answer 'I am the KICS bot and can answer only on questions related to the selected KICS result'.`

const assistantInput = `Checkmarx KICS has scanned this source code and reported the result.
This is the source code:
'<|KICS_SOURCE_START|>'
resource "aws_alb" "positive1" {
name               = "test-lb-tf"
internal           = false
load_balancer_type = "network"
subnets            = aws_subnet.public.*.id
enable_deletion_protection = false
tags = {
Environment = "production"
}
}
'<|KICS_SOURCE_END|>'
and this is the result (vulnerability or security issue) found by KICS:
'<|KICS_RESULT_START|>'
'ALB Deletion Protection Disabled' is detected in line 6 with severity 'LOW'.
'<|KICS_RESULT_END|>'`

const userInput = `The user question is:
'<|KICS_QUESTION_START|>'
"%s"
'<|KICS_QUESTION_END|>'`

var userQuestions = []string{
	"Explain the found result",
	"What is the impact of this security issue?",
	"Are there any additional resources or examples to help me understand this issue better?",
	"Can you explain the severity levels of the result?",
	"How can I fix this issue in the source code?",
	"Does this code have any SAST vulnerability?",
	"What should I check prior to fixing this?",
	"How can I validate that my fix is good?",
	"What should I know to eliminate such results in the future?",
}

func TestCallGPT(t *testing.T) {
	var history []message.Message
	var response []message.Message
	wrapper := NewStatelessWrapper(apikey, model.Model)
	for _, q := range userQuestions {
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
