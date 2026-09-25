// Package ollaya is a client for the ollaya /api/decide endpoint. It asks a
// decision model typed questions about some state and returns calibrated
// answers, similar to the System One Jev API.
package ollaya

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"within.website/x/web"
)

var (
	ErrNoModel           = errors.New("ollaya: no model specified")
	ErrNoState           = errors.New("ollaya: no state specified")
	ErrNoQuestions       = errors.New("ollaya: no questions specified")
	ErrNoInstructions    = errors.New("ollaya: question has no instructions")
	ErrNotEnoughCriteria = errors.New("ollaya: question needs at least two criteria")
	ErrUnknownType       = errors.New("ollaya: unknown question type")
)

// QuestionType is the kind of answer a Question wants.
type QuestionType string

const (
	// TypeChoice picks one named option out of a set.
	TypeChoice QuestionType = "choice"
	// TypeScore places the state on an ordered scale.
	TypeScore QuestionType = "score"
	// TypeNoul is the probability that a statement is true.
	TypeNoul QuestionType = "noul"
)

// Question is one question to ask about the state. Build it with Choice,
// Score, or Noul.
type Question struct {
	Type         QuestionType
	Instructions string

	choices map[string]string
	scale   []string
}

// Choice asks the model to pick one of the named options. The keys of
// criteria are the option names and the values describe each option.
func Choice(instructions string, criteria map[string]string) Question {
	return Question{
		Type:         TypeChoice,
		Instructions: instructions,
		choices:      criteria,
	}
}

// Score asks the model to place the state on an ordered scale. The levels go
// from lowest (index 0) to highest.
func Score(instructions string, levels ...string) Question {
	return Question{
		Type:         TypeScore,
		Instructions: instructions,
		scale:        levels,
	}
}

// Noul asks the model how likely it is that the instructions are true for
// the state.
func Noul(instructions string) Question {
	return Question{
		Type:         TypeNoul,
		Instructions: instructions,
	}
}

// Valid checks that the question is well-formed.
func (q Question) Valid() error {
	var errs []error

	if q.Instructions == "" {
		errs = append(errs, ErrNoInstructions)
	}

	switch q.Type {
	case TypeChoice:
		if len(q.choices) < 2 {
			errs = append(errs, ErrNotEnoughCriteria)
		}
	case TypeScore:
		if len(q.scale) < 2 {
			errs = append(errs, ErrNotEnoughCriteria)
		}
	case TypeNoul:
	default:
		errs = append(errs, fmt.Errorf("%w: %q", ErrUnknownType, q.Type))
	}

	return errors.Join(errs...)
}

func (q Question) MarshalJSON() ([]byte, error) {
	type wire struct {
		Type         QuestionType `json:"type"`
		Instructions string       `json:"instructions"`
		Criteria     any          `json:"criteria,omitempty"`
	}

	w := wire{Type: q.Type, Instructions: q.Instructions}
	switch q.Type {
	case TypeChoice:
		w.Criteria = q.choices
	case TypeScore:
		w.Criteria = q.scale
	}

	return json.Marshal(w)
}

// DecideRequest is the body of a /api/decide call.
type DecideRequest struct {
	Model     string              `json:"model"`
	State     string              `json:"state"`
	Questions map[string]Question `json:"questions"`
}

// Valid checks that the request is well-formed.
func (dr DecideRequest) Valid() error {
	var errs []error

	if dr.Model == "" {
		errs = append(errs, ErrNoModel)
	}

	if dr.State == "" {
		errs = append(errs, ErrNoState)
	}

	if len(dr.Questions) == 0 {
		errs = append(errs, ErrNoQuestions)
	}

	for name, q := range dr.Questions {
		if err := q.Valid(); err != nil {
			errs = append(errs, fmt.Errorf("question %q: %w", name, err))
		}
	}

	return errors.Join(errs...)
}

// Answer is the model's answer to one Question. Which fields are set depends
// on Type.
type Answer struct {
	Type QuestionType `json:"type"`

	// Choice is the winning option name. Set for TypeChoice.
	Choice string `json:"choice,omitempty"`

	// Score is the probability-weighted position on the scale, so it can land
	// between levels. Set for TypeScore.
	Score float64 `json:"score,omitempty"`

	// Legend maps scale indices ("0", "1", ...) to level text. Set for
	// TypeScore.
	Legend map[string]string `json:"legend,omitempty"`

	// Noul is the probability that the statement is true. Set for TypeNoul.
	// There is no separate confidence; the probability is the confidence.
	Noul float64 `json:"noul,omitempty"`

	// Confidence is how sure the model is. Set for TypeChoice and TypeScore.
	Confidence float64 `json:"confidence,omitempty"`

	// Probabilities maps option names (TypeChoice) or scale indices
	// (TypeScore) to their probability.
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
}

// Usage is the token accounting for a request.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Routing describes how the server routed the request to a model.
type Routing struct {
	Router string `json:"router"`
	Model  string `json:"model"`
	Route  string `json:"route"`
	Reason string `json:"reason"`
}

// DecideResponse is the result of a /api/decide call.
type DecideResponse struct {
	Model          string            `json:"model"`
	Answers        map[string]Answer `json:"answers"`
	Usage          Usage             `json:"usage"`
	Routing        *Routing          `json:"routing,omitempty"`
	StateTruncated bool              `json:"state_truncated"`
	DoneReason     string            `json:"done_reason"`
	CreatedAt      time.Time         `json:"created_at"`
	TotalDuration  time.Duration     `json:"total_duration"`
	LoadDuration   time.Duration     `json:"load_duration"`
	EvalDuration   time.Duration     `json:"eval_duration"`
}

// Client talks to an ollaya server.
type Client struct {
	baseURL string

	// HTTPClient is used for requests. If nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// NewClient creates a Client for the server at baseURL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
	}
}

// NewLocalClient creates a Client for a server on localhost:11435.
func NewLocalClient() *Client {
	return NewClient("http://localhost:11435")
}

// Decide asks the model the questions in inp about inp.State.
func (c *Client) Decide(ctx context.Context, inp *DecideRequest) (*DecideResponse, error) {
	if err := inp.Valid(); err != nil {
		return nil, fmt.Errorf("ollaya: invalid request: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(inp); err != nil {
		return nil, fmt.Errorf("ollaya: error encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/decide", buf)
	if err != nil {
		return nil, fmt.Errorf("ollaya: error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	cli := c.HTTPClient
	if cli == nil {
		cli = http.DefaultClient
	}

	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollaya: error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result DecideResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollaya: error decoding response: %w", err)
	}

	return &result, nil
}
