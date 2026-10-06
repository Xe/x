// Package systemone provides a client and HTTP handler for the System One API.
package systemone

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"within.website/x/web"
)

// QuestionType identifies one of the three System One primitives.
type QuestionType string

const (
	Noul   QuestionType = "noul"
	Choice QuestionType = "choice"
	Score  QuestionType = "score"
)

// Question describes a decision about the request's state. Instructions may be
// text or structured JSON. Criteria is optional for Noul, a map of named
// options for Choice, or an ordered slice of levels for Score.
type Question struct {
	Type         QuestionType `json:"type"`
	Instructions any          `json:"instructions,omitempty"`
	Criteria     any          `json:"criteria,omitempty"`
}

// Request evaluates one state against one or more named questions. State may
// be a string, JSON object, or JSON array.
type Request struct {
	Model     string              `json:"model"`
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
}

// Answer holds the fields for one of the three response types. The pointer
// fields retain valid zero values when an answer is marshaled back to JSON.
type Answer struct {
	Type          QuestionType       `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Legend        map[string]any     `json:"legend,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
}

// Usage reports token counts returned by the model.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Response contains one answer under each question's name.
type Response struct {
	Model   string            `json:"model"`
	Answers map[string]Answer `json:"answers"`
	Usage   Usage             `json:"usage"`
}

// Client calls a System One endpoint. Set APIKey for hosted TypeSafe requests;
// local compatible endpoints may not need one. HTTPClient defaults to
// http.DefaultClient when nil.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient returns a client for baseURL, such as
// https://api.typesafe.ai or http://localhost:11434.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/")}
}

// Evaluate sends a POST to /v1/systemone.
func (c *Client) Evaluate(ctx context.Context, input *Request) (*Response, error) {
	if input == nil {
		return nil, errors.New("systemone: nil request")
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if c == nil || c.BaseURL == "" {
		return nil, errors.New("systemone: base URL is required")
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(input); err != nil {
		return nil, fmt.Errorf("systemone: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/systemone", &body)
	if err != nil {
		return nil, fmt.Errorf("systemone: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("systemone: send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("systemone: decode response: %w", err)
	}
	return &result, nil
}

// Validate checks the request fields and question criteria before a call or
// before invoking a handler's evaluator.
func (r *Request) Validate() error {
	if r == nil {
		return errors.New("systemone: nil request")
	}
	if r.Model == "" {
		return errors.New("systemone: model is required")
	}
	state, err := json.Marshal(r.State)
	if err != nil {
		return fmt.Errorf("systemone: invalid state: %w", err)
	}
	if len(state) == 0 || (state[0] != '"' && state[0] != '{' && state[0] != '[') {
		return errors.New("systemone: state must be a string, object, or array")
	}
	if len(r.Questions) == 0 {
		return errors.New("systemone: at least one question is required")
	}
	for name, question := range r.Questions {
		if name == "" {
			return errors.New("systemone: question name is empty")
		}
		if err := question.validate(); err != nil {
			return fmt.Errorf("systemone: question %q: %w", name, err)
		}
	}
	return nil
}

func (q Question) validate() error {
	if q.Instructions != nil {
		if err := validateEntry(q.Instructions, false); err != nil {
			return fmt.Errorf("instructions: %w", err)
		}
	}
	switch q.Type {
	case Noul:
		if q.Criteria == nil {
			return nil
		}
		var criteria map[string]json.RawMessage
		if err := decodeCriteria(q.Criteria, &criteria); err != nil || criteria == nil {
			return errors.New("noul criteria must be an object")
		}
		for name, description := range criteria {
			if name != "true" && name != "false" {
				return fmt.Errorf("unknown noul criterion %q", name)
			}
			if err := validateEntry(description, true); err != nil {
				return fmt.Errorf("criterion %q: %w", name, err)
			}
		}
	case Choice:
		var criteria map[string]json.RawMessage
		if err := decodeCriteria(q.Criteria, &criteria); err != nil || criteria == nil {
			return errors.New("choice criteria must be an object")
		}
		if len(criteria) == 0 || len(criteria) > 255 {
			return errors.New("choice criteria must have 1 to 255 options")
		}
		for name, description := range criteria {
			if name == "" {
				return errors.New("choice option name is empty")
			}
			if err := validateEntry(description, true); err != nil {
				return fmt.Errorf("criterion %q: %w", name, err)
			}
		}
	case Score:
		var criteria []json.RawMessage
		if err := decodeCriteria(q.Criteria, &criteria); err != nil || criteria == nil {
			return errors.New("score criteria must be an array")
		}
		if len(criteria) < 2 || len(criteria) > 10 {
			return errors.New("score criteria must have 2 to 10 levels")
		}
		for i, description := range criteria {
			if err := validateEntry(description, true); err != nil {
				return fmt.Errorf("criterion %d: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("unknown type %q", q.Type)
	}
	return nil
}

func decodeCriteria(value any, target any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func validateEntry(value any, allowNull bool) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("must be a string, object, or array")
	}
	if data[0] == '"' || data[0] == '{' || data[0] == '[' || (allowNull && bytes.Equal(data, []byte("null"))) {
		return nil
	}
	return errors.New("must be a string, object, or array")
}

// Evaluator is implemented by Client and by local System One backends.
type Evaluator interface {
	Evaluate(context.Context, *Request) (*Response, error)
}

// EvaluatorFunc adapts a function to Evaluator.
type EvaluatorFunc func(context.Context, *Request) (*Response, error)

func (f EvaluatorFunc) Evaluate(ctx context.Context, req *Request) (*Response, error) {
	return f(ctx, req)
}

// NewHandler serves POST /v1/systemone using evaluator. Mount it at that path
// with an http.ServeMux.
func NewHandler(evaluator Evaluator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "method must be POST")
			return
		}
		if evaluator == nil {
			writeError(w, http.StatusInternalServerError, "evaluator is not configured")
			return
		}
		var input Request
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&input); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid JSON request: "+err.Error())
			return
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			writeError(w, http.StatusUnprocessableEntity, "request must contain one JSON object")
			return
		}
		if err := input.Validate(); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		output, err := evaluator.Evaluate(r.Context(), &input)
		if err != nil {
			var upstream *web.Error
			if errors.As(err, &upstream) {
				if json.Valid([]byte(upstream.ResponseBody)) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(upstream.GotStatus)
					_, _ = io.WriteString(w, upstream.ResponseBody)
				} else {
					writeError(w, upstream.GotStatus, upstream.ResponseBody)
				}
			} else {
				writeError(w, http.StatusBadGateway, err.Error())
			}
			return
		}
		if output == nil {
			writeError(w, http.StatusBadGateway, "evaluator returned no response")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(output); err != nil {
			return
		}
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}
