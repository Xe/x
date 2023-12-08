package ollama

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"time"
)

type StatusError struct {
	StatusCode   int
	Status       string
	ErrorMessage string `json:"error"`
}

func (e StatusError) Error() string {
	switch {
	case e.Status != "" && e.ErrorMessage != "":
		return fmt.Sprintf("%s: %s", e.Status, e.ErrorMessage)
	case e.Status != "":
		return e.Status
	case e.ErrorMessage != "":
		return e.ErrorMessage
	default:
		// this should not happen
		return "something went wrong, please see the ollama server logs for details"
	}
}

type GenerateRequest struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	System   string `json:"system"`
	Template string `json:"template"`
	Context  []int  `json:"context,omitempty"`
	Stream   *bool  `json:"stream,omitempty"`
	Raw      bool   `json:"raw,omitempty"`
	Format   string `json:"format"`

	Options map[string]interface{} `json:"options"`
}

// Options specfied in GenerateRequest, if you add a new option here add it to the API docs also
type Options struct {
	Runner

	// Predict options used at runtime
	NumKeep          int      `json:"num_keep,omitempty"`
	Seed             int      `json:"seed,omitempty"`
	NumPredict       int      `json:"num_predict,omitempty"`
	TopK             int      `json:"top_k,omitempty"`
	TopP             float32  `json:"top_p,omitempty"`
	TFSZ             float32  `json:"tfs_z,omitempty"`
	TypicalP         float32  `json:"typical_p,omitempty"`
	RepeatLastN      int      `json:"repeat_last_n,omitempty"`
	Temperature      float32  `json:"temperature,omitempty"`
	RepeatPenalty    float32  `json:"repeat_penalty,omitempty"`
	PresencePenalty  float32  `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32  `json:"frequency_penalty,omitempty"`
	Mirostat         int      `json:"mirostat,omitempty"`
	MirostatTau      float32  `json:"mirostat_tau,omitempty"`
	MirostatEta      float32  `json:"mirostat_eta,omitempty"`
	PenalizeNewline  bool     `json:"penalize_newline,omitempty"`
	Stop             []string `json:"stop,omitempty"`
}

// Runner options which must be set when the model is loaded into memory
type Runner struct {
	UseNUMA            bool    `json:"numa,omitempty"`
	NumCtx             int     `json:"num_ctx,omitempty"`
	NumBatch           int     `json:"num_batch,omitempty"`
	NumGQA             int     `json:"num_gqa,omitempty"`
	NumGPU             int     `json:"num_gpu,omitempty"`
	MainGPU            int     `json:"main_gpu,omitempty"`
	LowVRAM            bool    `json:"low_vram,omitempty"`
	F16KV              bool    `json:"f16_kv,omitempty"`
	LogitsAll          bool    `json:"logits_all,omitempty"`
	VocabOnly          bool    `json:"vocab_only,omitempty"`
	UseMMap            bool    `json:"use_mmap,omitempty"`
	UseMLock           bool    `json:"use_mlock,omitempty"`
	EmbeddingOnly      bool    `json:"embedding_only,omitempty"`
	RopeFrequencyBase  float32 `json:"rope_frequency_base,omitempty"`
	RopeFrequencyScale float32 `json:"rope_frequency_scale,omitempty"`
	NumThread          int     `json:"num_thread,omitempty"`
}

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`

	Options map[string]interface{} `json:"options"`
}

type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

type CreateRequest struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Modelfile string `json:"modelfile"`
	Stream    *bool  `json:"stream,omitempty"`
}

type DeleteRequest struct {
	Name string `json:"name"`
}

type ShowRequest struct {
	Name string `json:"name"`
}

type ShowResponse struct {
	License    string `json:"license,omitempty"`
	Modelfile  string `json:"modelfile,omitempty"`
	Parameters string `json:"parameters,omitempty"`
	Template   string `json:"template,omitempty"`
	System     string `json:"system,omitempty"`
}

type CopyRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type PullRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	Stream   *bool  `json:"stream,omitempty"`
}

type ProgressResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

type PushRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	Stream   *bool  `json:"stream,omitempty"`
}

type ListResponse struct {
	Models []ModelResponse `json:"models"`
}

type ModelResponse struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type GenerateResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`

	Done    bool  `json:"done"`
	Context []int `json:"context,omitempty"`

	TotalDuration      time.Duration `json:"total_duration,omitempty"`
	LoadDuration       time.Duration `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       time.Duration `json:"eval_duration,omitempty"`
}

func (r *GenerateResponse) Summary() {
	if r.TotalDuration > 0 {
		fmt.Fprintf(os.Stderr, "total duration:       %v\n", r.TotalDuration)
	}

	if r.LoadDuration > 0 {
		fmt.Fprintf(os.Stderr, "load duration:        %v\n", r.LoadDuration)
	}

	if r.PromptEvalCount > 0 {
		fmt.Fprintf(os.Stderr, "prompt eval count:    %d token(s)\n", r.PromptEvalCount)
	}

	if r.PromptEvalDuration > 0 {
		fmt.Fprintf(os.Stderr, "prompt eval duration: %s\n", r.PromptEvalDuration)
		fmt.Fprintf(os.Stderr, "prompt eval rate:     %.2f tokens/s\n", float64(r.PromptEvalCount)/r.PromptEvalDuration.Seconds())
	}

	if r.EvalCount > 0 {
		fmt.Fprintf(os.Stderr, "eval count:           %d token(s)\n", r.EvalCount)
	}

	if r.EvalDuration > 0 {
		fmt.Fprintf(os.Stderr, "eval duration:        %s\n", r.EvalDuration)
		fmt.Fprintf(os.Stderr, "eval rate:            %.2f tokens/s\n", float64(r.EvalCount)/r.EvalDuration.Seconds())
	}
}

var ErrInvalidOpts = fmt.Errorf("invalid options")

func (opts *Options) FromMap(m map[string]interface{}) error {
	valueOpts := reflect.ValueOf(opts).Elem() // names of the fields in the options struct
	typeOpts := reflect.TypeOf(opts).Elem()   // types of the fields in the options struct

	// build map of json struct tags to their types
	jsonOpts := make(map[string]reflect.StructField)
	for _, field := range reflect.VisibleFields(typeOpts) {
		jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonTag != "" {
			jsonOpts[jsonTag] = field
		}
	}

	invalidOpts := []string{}
	for key, val := range m {
		if opt, ok := jsonOpts[key]; ok {
			field := valueOpts.FieldByName(opt.Name)
			if field.IsValid() && field.CanSet() {
				if val == nil {
					continue
				}

				switch field.Kind() {
				case reflect.Int:
					switch t := val.(type) {
					case int64:
						field.SetInt(t)
					case float64:
						// when JSON unmarshals numbers, it uses float64, not int
						field.SetInt(int64(t))
					default:
						return fmt.Errorf("option %q must be of type integer", key)
					}
				case reflect.Bool:
					val, ok := val.(bool)
					if !ok {
						return fmt.Errorf("option %q must be of type boolean", key)
					}
					field.SetBool(val)
				case reflect.Float32:
					// JSON unmarshals to float64
					val, ok := val.(float64)
					if !ok {
						return fmt.Errorf("option %q must be of type float32", key)
					}
					field.SetFloat(val)
				case reflect.String:
					val, ok := val.(string)
					if !ok {
						return fmt.Errorf("option %q must be of type string", key)
					}
					field.SetString(val)
				case reflect.Slice:
					// JSON unmarshals to []interface{}, not []string
					val, ok := val.([]interface{})
					if !ok {
						return fmt.Errorf("option %q must be of type array", key)
					}
					// convert []interface{} to []string
					slice := make([]string, len(val))
					for i, item := range val {
						str, ok := item.(string)
						if !ok {
							return fmt.Errorf("option %q must be of an array of strings", key)
						}
						slice[i] = str
					}
					field.Set(reflect.ValueOf(slice))
				default:
					return fmt.Errorf("unknown type loading config params: %v", field.Kind())
				}
			}
		} else {
			invalidOpts = append(invalidOpts, key)
		}
	}

	if len(invalidOpts) > 0 {
		return fmt.Errorf("%w: %v", ErrInvalidOpts, strings.Join(invalidOpts, ", "))
	}
	return nil
}

func DefaultOptions() Options {
	return Options{
		// options set on request to runner
		NumPredict:       -1,
		NumKeep:          0,
		Temperature:      0.8,
		TopK:             40,
		TopP:             0.9,
		TFSZ:             1.0,
		TypicalP:         1.0,
		RepeatLastN:      64,
		RepeatPenalty:    1.1,
		PresencePenalty:  0.0,
		FrequencyPenalty: 0.0,
		Mirostat:         0,
		MirostatTau:      5.0,
		MirostatEta:      0.1,
		PenalizeNewline:  true,
		Seed:             -1,

		Runner: Runner{
			// options set when the model is loaded
			NumCtx:             2048,
			RopeFrequencyBase:  10000.0,
			RopeFrequencyScale: 1.0,
			NumBatch:           512,
			NumGPU:             -1, // -1 here indicates that NumGPU should be set dynamically
			NumGQA:             1,
			NumThread:          0, // let the runtime decide
			LowVRAM:            false,
			F16KV:              true,
			UseMLock:           false,
			UseMMap:            true,
			UseNUMA:            false,
			EmbeddingOnly:      true,
		},
	}
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	d.Duration = 5 * time.Minute

	switch t := v.(type) {
	case float64:
		if t < 0 {
			t = math.MaxFloat64
		}

		d.Duration = time.Duration(t)
	case string:
		d.Duration, err = time.ParseDuration(t)
		if err != nil {
			return err
		}
	}

	return nil
}
