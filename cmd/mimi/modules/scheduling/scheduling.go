/*
Package scheduling provides the scheduling module for Mimi.

This module allows users to CC Anise Robòta in their emails to schedule meetings and events.
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
	"net/http"
	"text/template"
	"time"

	"within.website/x/cmd/mimi/internal"
	"within.website/x/web/ollama"
)

func p[T any](t T) *T {
	return &t
}

const timeFormat string = "Monday January 2, 2006 at 3:04 PM"

//go:embed nise_template.txt
var niseTemplate string

type ConversationMember struct {
	Role  *string `json:"role,omitempty"`
	Name  string  `json:"name"`
	Email string  `json:"email"`
}

type NiseRequest struct {
	Month               string               `json:"month"`
	ConversationMembers []ConversationMember `json:"conversation_members"`
	Message             string               `json:"message"`
	Date                string               `json:"date"`
}

type NiseResponse struct {
	StartTime string               `json:"start_time"`
	Duration  string               `json:"duration"`
	Summary   string               `json:"summary"`
	Attendees []ConversationMember `json:"attendees"`
	Location  *string              `json:"location,omitempty"`
}

type Module struct {
	cli   *ollama.Client
	model string
}

func New() *Module {
	return &Module{
		cli:   internal.OllamaClient(),
		model: internal.OllamaModel(),
	}
}

func (m *Module) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	now := time.Now()

	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "scheduling: only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	dateString := now.Format(timeFormat)
	monthName := now.Month().String()

	var niseReq NiseRequest
	if err := json.NewDecoder(r.Body).Decode(&niseReq); err != nil {
		http.Error(w, fmt.Sprintf("scheduling: error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	niseReq.Date = dateString
	niseReq.Month = monthName

	resp, err := m.Handle(r.Context(), &niseReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("scheduling: error handling request: %v", err), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("scheduling: error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (m *Module) Handle(ctx context.Context, req *NiseRequest) (*NiseResponse, error) {
	tmpl := template.Must(template.New("nise").Parse(niseTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, req); err != nil {
		return nil, fmt.Errorf("scheduling: error executing template: %w", err)
	}

	resp, err := m.cli.Chat(ctx, &ollama.CompleteRequest{
		Model: m.model,
		Messages: []ollama.Message{
			{
				Role: "system",
				Content: `You are Anise Robòta, a scheduling assistant. You have been CCed in an email to schedule a meeting. Your task is to read this email and extract the following information into a JSON object:

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

	var niseResp NiseResponse

	if err := json.Unmarshal([]byte(resp.Message.Content), &niseResp); err != nil {
		return nil, fmt.Errorf("scheduling: error unmarshaling response: %w", err)
	}

	return &niseResp, nil
}
