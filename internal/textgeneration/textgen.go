package textgen

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"within.website/x/web"
)

var (
	tgServerURL = flag.String("textgen-server-url", "http://ontos:7860", "URL for the Stable Diffusion API")

	//go:embed data/characters
	characters embed.FS

	//go:embed data/presets
	presets embed.FS
)

func buildURL(path string) (*url.URL, error) {
	u, err := url.Parse(*tgServerURL)
	if err != nil {
		return nil, err
	}

	u.Path = path

	return u, nil
}

type Character struct {
	CharName        string `json:"char_name"`
	CharPersona     string `json:"char_persona"`
	CharGreeting    string `json:"char_greeting"`
	WorldScenario   string `json:"world_scenario"`
	ExampleDialogue string `json:"example_dialogue"`
}

/*
  [
       : string, // represents text string of 'Input' Textbox component
       : number, // represents selected value of 'max_new_tokens' Slider component
       : boolean, // represents checked status of 'do_sample' Checkbox component
       : number, // represents selected value of 'temperature' Slider component
       : number, // represents selected value of 'top_p' Slider component
       : number, // represents selected value of 'typical_p' Slider component
       : number, // represents selected value of 'repetition_penalty' Slider component
       : number, // represents selected value of 'top_k' Slider component
       : number, // represents selected value of 'min_length' Slider component
       : number, // represents selected value of 'no_repeat_ngram_size' Slider component
       : number, // represents selected value of 'num_beams' Slider component
       : number, // represents selected value of 'penalty_alpha' Slider component
       : number, // represents selected value of 'length_penalty' Slider component
       : boolean, // represents checked status of 'early_stopping' Checkbox component
       : string, // represents text string of 'Your name' Textbox component
       : string, // represents text string of 'Bot's name' Textbox component
       : string, // represents text string of 'Context' Textbox component
       : boolean, // represents checked status of 'Stop generating at new line character?' Checkbox component
       : number, // represents selected value of 'Maximum prompt size in tokens' Slider component
       : number, // represents selected value of 'Generation attempts (for longer replies)' Slider component
  ]
*/

type ChatRequest struct {
	Input              string  `json:"input"`
	MaxNewTokens       int     `json:"max_new_tokens"`
	DoSample           bool    `json:"do_sample"`
	Temp               float64 `json:"temperature"`
	TopP               float64 `json:"top_p"`
	TypicalP           float64 `json:"typical_p"`
	RepetitionPenalty  float64 `json:"repetition_penalty"`
	TopK               float64 `json:"top_k"`
	MinLength          int     `json:"min_length"`
	NoRepeatNgramSize  int     `json:"no_repeat_ngram_size"`
	NumBeams           int     `json:"num_beams"`
	PenaltyAlpha       float64 `json:"penalty_alpha"`
	LengthPenalty      float64 `json:"length_penalty"`
	EarlyStopping      bool    `json:"early_stopping"`
	YourName           string  `json:"your_name"`
	BotName            string  `json:"bot_name"`
	Context            string  `json:"context"`
	StopAfterNewline   bool    `json:"stop_after_newline"`
	MaxPromptSize      int     `json:"max_prompt_size"`
	GenerationAttempts int     `json:"generation_attempts"`
}

func (cr *ChatRequest) ApplyCharacter(name string) error {
	fin, err := characters.Open("data/characters/" + name + ".json")
	if err != nil {
		return fmt.Errorf("textgen: can't open character %s: %w", name, err)
	}
	defer fin.Close()

	var ch Character
	if err := json.NewDecoder(fin).Decode(&ch); err != nil {
		return fmt.Errorf("textgen: can't decode character %s: %w", name, err)
	}

	cr.BotName = ch.CharName

	var sb strings.Builder

	fmt.Fprintln(&sb, ch.CharPersona)
	fmt.Fprintln(&sb, "<START>")
	fmt.Fprintln(&sb, ch.ExampleDialogue)
	fmt.Fprintln(&sb)

	cr.Context = sb.String()

	return nil
}

// ApplyPreset mutates cr with the details in the preset by name.
func (cr *ChatRequest) ApplyPreset(name string) error {
	finData, err := presets.ReadFile("data/presets/" + name + ".txt")
	if err != nil {
		return fmt.Errorf("textgen: can't open preset %s: %w", name, err)
	}

	var data = map[string]any{}

	for _, line := range strings.Split(string(finData), "\n") {
		if line == "" {
			break
		}

		kv := strings.SplitN(line, "=", 2)
		k, v := kv[0], kv[1]
		switch v {
		case "True":
			data[k] = true
		case "False":
			data[k] = false
		default:
			num, err := strconv.ParseFloat(v, 64)
			if err != nil {
				fmt.Errorf("textgen: can't parse %q as float64: %w", v, err)
			}

			data[k] = num
		}
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return fmt.Errorf("textgen: can't encode data to JSON: %w", err)
	}

	if err := json.Unmarshal(buf.Bytes(), cr); err != nil {
		return fmt.Errorf("textgeneration: can't decode data to ChatRequest: %w", err)
	}

	return nil
}

func (cr *ChatRequest) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(struct {
		Data []any `json:"data"`
	}{
		Data: []any{
			cr.Input,
			cr.MaxNewTokens,
			cr.DoSample,
			cr.Temp,
			cr.TopP,
			cr.TypicalP,
			cr.RepetitionPenalty,
			cr.TopK,
			cr.MinLength,
			cr.NoRepeatNgramSize,
			cr.NumBeams,
			cr.PenaltyAlpha,
			cr.LengthPenalty,
			cr.EarlyStopping,
			// cr.YourName,
			// cr.BotName,
			// cr.Context,
			// cr.StopAfterNewline,
			// cr.MaxPromptSize,
			// cr.GenerationAttempts,
		}}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type ChatResponse struct {
	Data         []string `json:"data"` // [0] is user input, [1] is bot output
	Duration     float64  `json:"duration"`
	IsGenerating bool     `json:"is_generating"`
}

var (
	Default *Client = &Client{
		HTTP: http.DefaultClient,
	}
)

func Generate(ctx context.Context, inp *ChatRequest) (*ChatResponse, error) {
	return Default.Generate(ctx, inp)
}

type Client struct {
	HTTP *http.Client
}

func (c *Client) Generate(ctx context.Context, cr *ChatRequest) (*ChatResponse, error) {
	u, err := buildURL("/run/textgen")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(cr); err != nil {
		return nil, err
	}

	log.Println(buf.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't fetch response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	buf = bytes.Buffer{}
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("can't read body")
	}

	log.Println(buf.String())

	var result ChatResponse
	if err := json.NewDecoder(&buf).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing ChatResponse: %w", err)
	}

	return &result, nil
}
