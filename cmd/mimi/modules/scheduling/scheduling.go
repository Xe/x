/*
Package scheduling provides the scheduling module for Mimi.

This module allows authorized users to CC Nise in their emails to schedule meetings and events.
When Nise is CCed in an email, she will parse the email and extract the date, time, and
location of the event. Nise will then create a Google Calendar event and send an invitation.
*/
package scheduling

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"text/template"
	"time"

	"github.com/cenkalti/backoff/v4"
	"within.website/x/cmd/mimi/internal"
	"within.website/x/web/ollama"
)

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --twirp_out=. scheduling.proto

func p[T any](t T) *T {
	return &t
}

const timeFormat string = "Monday January 2, 2006 at 3:04 PM"

//go:embed nise_template.txt
var niseTemplate string

type Module struct {
	cli   *ollama.Client
	model string

	UnimplementedSchedulingServer
}

func New() *Module {
	return &Module{
		cli:   internal.OllamaClient(),
		model: internal.OllamaModel(),
	}
}

func (m *Module) ParseEmail(ctx context.Context, req *ParseReq) (*ParseResp, error) {
	bo := backoff.NewExponentialBackOff()

	return backoff.RetryNotifyWithData[*ParseResp](func() (*ParseResp, error) {
		return m.parseEmail(ctx, req)
	}, bo, func(err error, t time.Duration) {
		slog.Error("error parsing email", "err", err, "t", t.String())
	})
}

func (m *Module) parseEmail(ctx context.Context, req *ParseReq) (*ParseResp, error) {
	tmpl := template.Must(template.New("nise").Parse(niseTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, req); err != nil {
		return nil, backoff.Permanent(fmt.Errorf("scheduling: error executing template: %w", err))
	}

	resp, err := m.cli.Chat(ctx, &ollama.CompleteRequest{
		Model: m.model,
		Messages: []ollama.Message{
			{
				Role: "system",
				Content: `You are Nise, a scheduling assistant. You have been CCed in an email to schedule a meeting. Your task is to read this email and extract the following information into a JSON object:

				* The start time of the meeting
				* The duration of the meeting
				* A summary of the meeting
				* The attendees in the meeting with their name and email address
				* If a location is given, add it as an optional field named "location"
				
				The JSON object should be formatted like this:
				
				{"start_time": "Monday April 25, 2024 at 12:45 PM", duration: "30m", "summary": "A brief summary of the email", "attendees": [{"name": "From address name", "email": "from_user@domain.tld"}, {"name": "Any CC names", "email": "cc_user@domain.tld"]}`,
			},
			{
				Role:    "user",
				Content: buf.String(),
			},
		},
		Format: p("json"),
		Stream: false,
	})
	if err != nil {
		return nil, fmt.Errorf("scheduling: error summarizing email: %w", err)
	}

	var niseResp ParseResp

	if err := json.Unmarshal([]byte(resp.Message.Content), &niseResp); err != nil {
		return nil, fmt.Errorf("scheduling: error unmarshaling response: %w", err)
	}

	return &niseResp, nil
}
