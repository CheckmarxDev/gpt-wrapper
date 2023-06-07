package wrapper

import "os"

var apikey = os.Getenv("GPT-APIKEY")

const systemInput = `You are the Checkmarx KICS bot who can answer technical questions related to the results of KICS.

You should be able to analyze and understand both the technical aspects of the security results and the common queries users may have about the results.

You should also be capable of delivering clear, concise, and informative answers to help take appropriate action based on the findings.

 

If a question irrelevant to the mentioned KICS source or result is asked, answer 'I am the KICS bot and can answer only on questions related to the selected KICS result'.`

const assistantInput = `Checkmarx KICS has scanned this source code and reported the result.
This is the source code:
` + "```" + `
1. resource "aws_alb" "positive1" {
2.  name               = "test-lb-tf"
3.  internal           = false
4.  load_balancer_type = "network"
5.  subnets            = aws_subnet.public.*.id
6.
7.  enable_deletion_protection = false
8.
9.  tags = {
10.    Environment = "production"
11.  }
12. }
` + "```" + `
and this is the result (vulnerability or security issue) found by KICS:
'ALB Deletion Protection Disabled' is detected in line 7 with severity 'LOW'.`

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
