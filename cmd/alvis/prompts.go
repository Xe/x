package main

import "within.website/x/web/openai/chatgpt"

const basePrompt = `You are Alvis, a large language model systems administrator.

You are tasked with responding to pagerduty pages and taking action to fix the services in question. You will call the functions given in order to fix issues.

If you get a fork/exec error, escalate.

Any other text you type will be sent to the pagerduty incident as a comment.`

const summaryPrompt = `You are Alvis, a large language model systems administrator.

You will be given the events of the incident. Use them to write your summary and postmortem.`

var functions = []chatgpt.Function{
	{
		Name:        "restart_fly_app",
		Description: "restarts an app running on fly.io, after running this you should run a health check",
		Parameters: chatgpt.Param{
			Type: "object",
			Properties: chatgpt.Properties{
				"app": {
					Type:        "string",
					Description: "the name of the app to restart",
				},
				"reason": {
					Type:        "string",
					Description: "the reason for restarting the app",
				},
			},
		},
	},
	{
		Name:        "perform_health_check",
		Description: "performs a health check on an app running on fly.io",
		Parameters: chatgpt.Param{
			Type:       "object",
			Properties: chatgpt.Properties{},
		},
	},
	{
		Name:        "wait",
		Description: "waits for a given amount of time before continuing",
		Parameters: chatgpt.Param{
			Type: "object",
			Properties: chatgpt.Properties{
				"duration": {
					Type:        "integer",
					Description: "the duration to wait for in minutes",
				},
			},
		},
	},
	{
		Name:        "close",
		Description: "closes the incident, you should only do this if you are sure the issue is resolved.",
		Parameters: chatgpt.Param{
			Type: "object",
			Properties: chatgpt.Properties{
				"reason": {
					Type:        "string",
					Description: "the reason for closing the incident",
				},
			},
		},
	},
	{
		Name:        "escalate",
		Description: "escalates the incident to the next level of support",
		Parameters: chatgpt.Param{
			Type:       "object",
			Properties: chatgpt.Properties{},
		},
	},
}
