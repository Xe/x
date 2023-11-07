package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"within.website/x/web"
)

type RunStatus string

const (
	RunStatusQueued         RunStatus = "queued"
	RunStatusInProgress     RunStatus = "in_progress"
	RunStatusRequiresAction RunStatus = "requires_action"
	RunStatusCancelling     RunStatus = "cancelling"
	RunStatusCancelled      RunStatus = "cancelled"
	RunStatusFailed         RunStatus = "failed"
	RunStatusCompleted      RunStatus = "completed"
	RunStatusExpired        RunStatus = "expired"
)

type RequiredAction struct {
	Type              string     `json:"type"`
	SubmitToolOutputs []ToolCall `json:"submit_tool_outputs"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments"`
}

type ToolOutput struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
}

type Run struct {
	ID           string            `json:"id"`
	Object       string            `json:"object"`
	CreatedAt    int64             `json:"created_at"`
	AssistantID  string            `json:"assistant_id"`
	ThreadID     string            `json:"thread_id"`
	Status       RunStatus         `json:"status"`
	StartedAt    int64             `json:"started_at"`
	ExpiresAt    *int64            `json:"expires_at,omitempty"`
	CancelledAt  *int64            `json:"cancelled_at,omitempty"`
	FailedAt     *int64            `json:"failed_at,omitempty"`
	LastError    *string           `json:"last_error,omitempty"`
	Model        string            `json:"model"`
	Instructions *string           `json:"instructions,omitempty"`
	Tools        []Tool            `json:"tools"`
	FileIDs      []string          `json:"file_ids"`
	Metadata     map[string]string `json:"metadata"`
}

func (c *Client) CreateRun(ctx context.Context, tid, aid string) (*Run, error) {
	type request struct {
		AssistantID string `json:"assistant_id"`
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(request{aid}); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+tid+"/runs", buf)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Run
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetRun(ctx context.Context, tid, rid string) (*Run, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+tid+"/runs/"+rid, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Run
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) CancelRun(ctx context.Context, tid, rid string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+tid+"/runs/"+rid+"/cancel", nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return web.NewError(http.StatusOK, resp)
	}

	return nil
}

func (c *Client) ListRuns(ctx context.Context, tid string) ([]Run, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+tid+"/runs", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result struct {
		Data []Run `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) SubmitToolOutput(ctx context.Context, tid, rid string, to ToolOutput) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(to); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+tid+"/runs/"+rid+"/submit_tool_output", buf)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return web.NewError(http.StatusOK, resp)
	}

	return nil
}
