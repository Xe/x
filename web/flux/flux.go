package flux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"within.website/x/web"
)

// Struct definitions based on the OpenAPI schema

type Input struct {
	Prompt            string  `json:"prompt"`
	Image             string  `json:"image,omitempty"`
	AspectRatio       string  `json:"aspect_ratio,omitempty"`
	NumOutputs        int     `json:"num_outputs"`
	GuidanceScale     float64 `json:"guidance_scale"`
	MaxSequenceLength int     `json:"max_sequence_length"`
	NumInferenceSteps int     `json:"num_inference_steps"`
	PromptStrength    float64 `json:"prompt_strength"`
	Seed              *int    `json:"seed,omitempty"`
	OutputFormat      string  `json:"output_format"`
	OutputQuality     int     `json:"output_quality"`
}

type Output []string

type PredictionRequest struct {
	Input               Input    `json:"input"`
	ID                  string   `json:"id"`
	CreatedAt           string   `json:"created_at"`
	OutputFilePrefix    string   `json:"output_file_prefix"`
	Webhook             string   `json:"webhook"`
	WebhookEventsFilter []string `json:"webhook_events_filter"`
}

type PredictionResponse struct {
	Input       Input                  `json:"input"`
	Output      Output                 `json:"output"`
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	CreatedAt   string                 `json:"created_at"`
	StartedAt   string                 `json:"started_at"`
	CompletedAt string                 `json:"completed_at"`
	Logs        string                 `json:"logs"`
	Error       string                 `json:"error"`
	Status      string                 `json:"status"`
	Metrics     map[string]interface{} `json:"metrics"`
}

type HTTPValidationError struct {
	Detail []ValidationError `json:"detail"`
}

type ValidationError struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
}

// HealthCheckResponse represents the response structure for the health check endpoint.
type HealthCheckResponse struct {
	Status string `json:"status"`
}

// Client struct

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new API client

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Methods to interact with the API endpoints

func (c *Client) Predict(predictionReq PredictionRequest) (*PredictionResponse, error) {
	body, err := json.Marshal(predictionReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/predictions", c.BaseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var predictionResp PredictionResponse
	err = json.NewDecoder(resp.Body).Decode(&predictionResp)
	if err != nil {
		return nil, err
	}

	return &predictionResp, nil
}

func (c *Client) PredictIdempotent(predictionID string, predictionReq PredictionRequest) (*PredictionResponse, error) {
	body, err := json.Marshal(predictionReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/predictions/%s", c.BaseURL, predictionID), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var predictionResp PredictionResponse
	err = json.NewDecoder(resp.Body).Decode(&predictionResp)
	if err != nil {
		return nil, err
	}

	return &predictionResp, nil
}

func (c *Client) CancelPrediction(predictionID string) (*http.Response, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/predictions/%s/cancel", c.BaseURL, predictionID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	return resp, nil
}

// HealthCheck checks the health of the service
func (c *Client) HealthCheck() (*HealthCheckResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/health-check", c.BaseURL), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(resp.StatusCode, resp)
	}

	var healthResp HealthCheckResponse
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	if err != nil {
		return nil, err
	}

	return &healthResp, nil
}
