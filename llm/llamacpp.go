package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"within.website/x/web"
)

type Client struct {
	cli       *http.Client
	serverURL string
}

func NewClient(cli *http.Client, serverURL string) *Client {
	return &Client{
		cli:       cli,
		serverURL: serverURL,
	}
}

func (c *Client) ExecSession(session Session) (*LLAMAResponse, error) {
	opts := DefaultLLAMAOpts()
	opts.Prompt = session.ChatML()
	return c.Predict(opts)
}

func (c *Client) Predict(opts *LLAMAOpts) (*LLAMAResponse, error) {
	jsonData, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	// Make a POST request to the server
	resp, err := c.cli.Post(c.serverURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result LLAMAResponse

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

type LLAMAOpts struct {
	Temperature   float64 `json:"temperature"`
	TopK          int     `json:"top_k"`
	TopP          float64 `json:"top_p"`
	Stream        bool    `json:"stream"`
	Prompt        string  `json:"prompt"`
	RepeatPenalty float64 `json:"repeat_penalty"`
	RepeatLastN   int     `json:"repeat_last_n"`
	Mirostat      int     `json:"mirostat"`
	NPredict      int     `json:"n_predict"`
}

func DefaultLLAMAOpts() *LLAMAOpts {
	return &LLAMAOpts{
		Temperature:   0.8,
		TopK:          40,
		TopP:          0.9,
		Stream:        false,
		RepeatPenalty: 1.15,
		RepeatLastN:   512,
		Mirostat:      2,
		NPredict:      2048,
	}
}

type LLAMAResponse struct {
	Content            string             `json:"content"`
	GenerationSettings GenerationSettings `json:"generation_settings"`
	Model              string             `json:"model"`
	Prompt             string             `json:"prompt"`
	Stop               bool               `json:"stop"`
	StoppedEos         bool               `json:"stopped_eos"`
	StoppedLimit       bool               `json:"stopped_limit"`
	StoppedWord        bool               `json:"stopped_word"`
	StoppingWord       string             `json:"stopping_word"`
	Timings            Timings            `json:"timings"`
	TokensCached       int                `json:"tokens_cached"`
	TokensEvaluated    int                `json:"tokens_evaluated"`
	TokensPredicted    int                `json:"tokens_predicted"`
	Truncated          bool               `json:"truncated"`
}

type GenerationSettings struct {
	FrequencyPenalty float64 `json:"frequency_penalty"`
	Grammar          string  `json:"grammar"`
	IgnoreEos        bool    `json:"ignore_eos"`
	LogitBias        []any   `json:"logit_bias"`
	Mirostat         int     `json:"mirostat"`
	MirostatEta      float64 `json:"mirostat_eta"`
	MirostatTau      float64 `json:"mirostat_tau"`
	Model            string  `json:"model"`
	NCtx             int     `json:"n_ctx"`
	NKeep            int     `json:"n_keep"`
	NPredict         int     `json:"n_predict"`
	NProbs           int     `json:"n_probs"`
	PenalizeNl       bool    `json:"penalize_nl"`
	PresencePenalty  float64 `json:"presence_penalty"`
	RepeatLastN      int     `json:"repeat_last_n"`
	RepeatPenalty    float64 `json:"repeat_penalty"`
	Seed             int64   `json:"seed"`
	Stop             []any   `json:"stop"`
	Stream           bool    `json:"stream"`
	Temp             float64 `json:"temp"`
	TfsZ             float64 `json:"tfs_z"`
	TopK             int     `json:"top_k"`
	TopP             float64 `json:"top_p"`
	TypicalP         float64 `json:"typical_p"`
}

type Timings struct {
	PredictedMs         float64 `json:"predicted_ms"`
	PredictedN          int     `json:"predicted_n"`
	PredictedPerSecond  float64 `json:"predicted_per_second"`
	PredictedPerTokenMs float64 `json:"predicted_per_token_ms"`
	PromptMs            float64 `json:"prompt_ms"`
	PromptN             int     `json:"prompt_n"`
	PromptPerSecond     float64 `json:"prompt_per_second"`
	PromptPerTokenMs    float64 `json:"prompt_per_token_ms"`
}

func (t Timings) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Float64("predicted_ms", t.PredictedMs),
		slog.Int("predicted_n", t.PredictedN),
		slog.Float64("predicted_per_second", t.PredictedPerSecond),
		slog.Float64("predicted_per_token_ms", t.PredictedPerTokenMs),
		slog.Float64("prompt_ms", t.PromptMs),
		slog.Int("prompt_n", t.PromptN),
		slog.Float64("prompt_per_second", t.PromptPerSecond),
		slog.Float64("prompt_per_token_ms", t.PromptPerTokenMs),
	)
}
