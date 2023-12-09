package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Request struct {
	Input string `json:"input"`
}

type Response struct {
	ID      string    `json:"id"`
	Model   string    `json:"model"`
	Results []Results `json:"results"`
}

func (r Response) Flagged() bool {
	var result bool

	for _, set := range r.Results {
		if set.Flagged {
			result = true
		}
	}

	return result
}

func (r Response) Reasons() string {
	if !r.Flagged() {
		return ""
	}

	var sb strings.Builder

	fmt.Fprintln(&sb, "Your request failed for the following reasons:")
	fmt.Fprintln(&sb)

	for _, set := range r.Results {
		if set.Categories.Hate {
			fmt.Fprintln(&sb, "- hate")
		}
		if set.Categories.HateThreatening {
			fmt.Fprintln(&sb, "- hate (threatening)")
		}
		if set.Categories.SelfHarm {
			fmt.Fprintln(&sb, "- self harm")
		}
		if set.Categories.Sexual {
			fmt.Fprintln(&sb, "- sexual")
		}
		if set.Categories.SexualMinors {
			fmt.Fprintln(&sb, "- sexual (minors)")
		}
		if set.Categories.Violence {
			fmt.Fprintln(&sb, "- violence")
		}
		if set.Categories.ViolenceGraphic {
			fmt.Fprintln(&sb, "- violence (graphic)")
		}
	}

	return sb.String()
}

type Categories struct {
	Hate            bool `json:"hate"`
	HateThreatening bool `json:"hate/threatening"`
	SelfHarm        bool `json:"self-harm"`
	Sexual          bool `json:"sexual"`
	SexualMinors    bool `json:"sexual/minors"`
	Violence        bool `json:"violence"`
	ViolenceGraphic bool `json:"violence/graphic"`
}
type CategoryScores struct {
	Hate            float64 `json:"hate"`
	HateThreatening float64 `json:"hate/threatening"`
	SelfHarm        float64 `json:"self-harm"`
	Sexual          float64 `json:"sexual"`
	SexualMinors    float64 `json:"sexual/minors"`
	Violence        float64 `json:"violence"`
	ViolenceGraphic float64 `json:"violence/graphic"`
}
type Results struct {
	Categories     Categories     `json:"categories"`
	CategoryScores CategoryScores `json:"category_scores"`
	Flagged        bool           `json:"flagged"`
}

type Client struct {
	httpCli *http.Client
	apiKey  string
}

func New(httpCli *http.Client, apiKey string) *Client {
	return &Client{
		httpCli: httpCli,
		apiKey:  apiKey,
	}
}

func (c Client) Check(ctx context.Context, input string) (*Response, error) {
	data := Request{
		Input: input,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/moderations", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
