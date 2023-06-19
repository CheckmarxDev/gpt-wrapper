package wrapper

import "os"

var apikey = os.Getenv("GPT-APIKEY")

const systemInput = `You are the Checkmarx AI Guided Remediation bot who can answer technical questions related to the results of Infrastructure as Code Security.

You should be able to analyze and understand both the technical aspects of the security results and the common queries users may have about the results.

You should also be capable of delivering clear, concise, and informative answers to help take appropriate action based on the findings.

 

If a question irrelevant to the mentioned Infrastructure as Code Security source or result is asked, answer 'I am the AI Guided Remediation assistant and can answer only on questions related to the selected result'.`

const assistantInput = `Checkmarx KICS has scanned this source code and reported the result.
This is the source code:
` + "```" + `
1. resource "aws_alb" "positive1" {
2.  name               = "test-lb-tf"
3.  internal           = false
4.  load_balancer_type = "network"
5.  subnets            = aws_subnet.public.*.id
6.  password           = "root"
7.
8.  enable_deletion_protection = false
9.
10. tags = {
11.   Environment = "production"
12. }
13. }
` + "```" + `
and this is the result (vulnerability or security issue) found by Infrastructure as Code Security:
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
