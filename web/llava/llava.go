package llava

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	"within.website/x/web"
)

type Image struct {
	Data []byte `json:"data"`
	ID   int    `json:"id"`
}

type Request struct {
	Stream           bool     `json:"stream"`
	NPredict         int      `json:"n_predict"`
	Temperature      float64  `json:"temperature"`
	Stop             []string `json:"stop"`
	RepeatLastN      int      `json:"repeat_last_n"`
	RepeatPenalty    float64  `json:"repeat_penalty"`
	TopK             int      `json:"top_k"`
	TopP             float64  `json:"top_p"`
	TfsZ             int      `json:"tfs_z"`
	TypicalP         int      `json:"typical_p"`
	PresencePenalty  int      `json:"presence_penalty"`
	FrequencyPenalty int      `json:"frequency_penalty"`
	Mirostat         int      `json:"mirostat"`
	MirostatTau      int      `json:"mirostat_tau"`
	MirostatEta      float64  `json:"mirostat_eta"`
	Grammar          string   `json:"grammar"`
	NProbs           int      `json:"n_probs"`
	ImageData        []Image  `json:"image_data"`
	CachePrompt      bool     `json:"cache_prompt"`
	SlotID           int      `json:"slot_id"`
	Prompt           string   `json:"prompt"`
}

var imageID = 10
var imageLock = sync.Mutex{}

func DefaultRequest(prompt string, image io.Reader) (*Request, error) {
	imageLock.Lock()
	defer imageLock.Unlock()

	imageID++

	imageData, err := io.ReadAll(image)
	if err != nil {
		return nil, err
	}

	return &Request{
		Stream:           false,
		NPredict:         400,
		Temperature:      0.7,
		Stop:             []string{"</s>", "Mimi:", "User:"},
		RepeatLastN:      256,
		RepeatPenalty:    1.18,
		TopK:             40,
		TopP:             0.5,
		TfsZ:             1,
		TypicalP:         1,
		PresencePenalty:  0,
		FrequencyPenalty: 0,
		Mirostat:         0,
		MirostatTau:      5,
		MirostatEta:      0.1,
		Grammar:          "",
		NProbs:           0,
		ImageData: []Image{
			{
				Data: imageData,
				ID:   imageID,
			},
		},
		CachePrompt: true,
		SlotID:      -1,
		Prompt:      formatPrompt(prompt, imageID),
	}, nil
}

func Describe(ctx context.Context, server string, req *Request) (*Response, error) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, server, &buf)
	if err != nil {
		return nil, err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("User-Agent", "within.website/x/llm/llava")

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("llava: http request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var llr Response
	if err := json.NewDecoder(resp.Body).Decode(&llr); err != nil {
		return nil, fmt.Errorf("llava: json decode error: %w", err)
	}

	return &llr, nil
}

func formatPrompt(prompt string, imageID int) string {
	const basePrompt = `A chat between a curious human and an artificial intelligence assistant. The assistant gives helpful, detailed, and polite answers to the human's questions.
	USER:[img-${imageID}]${prompt}
	ASSISTANT:`
	return os.Expand(basePrompt, func(key string) string {
		switch key {
		case "prompt":
			return prompt
		case "imageID":
			return strconv.Itoa(imageID)
		default:
			return ""
		}
	})
}

type Response struct {
	Content         string  `json:"content"`
	Model           string  `json:"model"`
	Prompt          string  `json:"prompt"`
	SlotID          int     `json:"slot_id"`
	Stop            bool    `json:"stop"`
	Timings         Timings `json:"timings"`
	TokensCached    int     `json:"tokens_cached"`
	TokensEvaluated int     `json:"tokens_evaluated"`
	TokensPredicted int     `json:"tokens_predicted"`
	Truncated       bool    `json:"truncated"`
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
