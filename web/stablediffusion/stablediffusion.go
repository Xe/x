package stablediffusion

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"within.website/x/web"
)

var (
	sdServerURL = flag.String("within.website/x/web/stablediffusion/server-url", "http://logos:7860", "URL for the Stable Diffusion API used with the default client")
)

func buildURL(base, path string) (*url.URL, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}

	u.Path = path

	return u, nil
}

// SimpleImageRequest is all of the parameters needed to generate an image.
type SimpleImageRequest struct {
	Prompt           string   `json:"prompt"`
	NegativePrompt   string   `json:"negative_prompt"`
	Styles           []string `json:"styles"`
	Seed             int      `json:"seed"`
	SamplerName      string   `json:"sampler_name"`
	BatchSize        int      `json:"batch_size"`
	NIter            int      `json:"n_iter"`
	Steps            int      `json:"steps"`
	CfgScale         int      `json:"cfg_scale"`
	Width            int      `json:"width"`
	Height           int      `json:"height"`
	SNoise           int      `json:"s_noise"`
	OverrideSettings struct {
	} `json:"override_settings"`
	OverrideSettingsRestoreAfterwards bool `json:"override_settings_restore_afterwards"`
}

type ImageResponse struct {
	Images [][]byte `json:"images"`
	Info   string   `json:"info"`
}

type ImageInfo struct {
	Prompt                string      `json:"prompt"`
	AllPrompts            []string    `json:"all_prompts"`
	NegativePrompt        string      `json:"negative_prompt"`
	AllNegativePrompts    []string    `json:"all_negative_prompts"`
	Seed                  int         `json:"seed"`
	AllSeeds              []int       `json:"all_seeds"`
	Subseed               int         `json:"subseed"`
	AllSubseeds           []int       `json:"all_subseeds"`
	SubseedStrength       int         `json:"subseed_strength"`
	Width                 int         `json:"width"`
	Height                int         `json:"height"`
	SamplerName           string      `json:"sampler_name"`
	CfgScale              float64     `json:"cfg_scale"`
	Steps                 int         `json:"steps"`
	BatchSize             int         `json:"batch_size"`
	RestoreFaces          bool        `json:"restore_faces"`
	FaceRestorationModel  interface{} `json:"face_restoration_model"`
	SdModelHash           string      `json:"sd_model_hash"`
	SeedResizeFromW       int         `json:"seed_resize_from_w"`
	SeedResizeFromH       int         `json:"seed_resize_from_h"`
	DenoisingStrength     int         `json:"denoising_strength"`
	ExtraGenerationParams struct {
	} `json:"extra_generation_params"`
	IndexOfFirstImage             int           `json:"index_of_first_image"`
	Infotexts                     []string      `json:"infotexts"`
	Styles                        []interface{} `json:"styles"`
	JobTimestamp                  string        `json:"job_timestamp"`
	ClipSkip                      int           `json:"clip_skip"`
	IsUsingInpaintingConditioning bool          `json:"is_using_inpainting_conditioning"`
}

var (
	Default *Client = &Client{
		HTTP: http.DefaultClient,
	}
	lock sync.Mutex
)

func Generate(ctx context.Context, inp SimpleImageRequest) (*ImageResponse, error) {
	lock.Lock()
	Default.APIServer = *sdServerURL
	lock.Unlock()
	return Default.Generate(ctx, inp)
}

type Client struct {
	HTTP      *http.Client
	APIServer string
}

func (c *Client) Generate(ctx context.Context, inp SimpleImageRequest) (*ImageResponse, error) {
	u, err := buildURL(c.APIServer, "/sdapi/v1/txt2img")
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(inp); err != nil {
		return nil, fmt.Errorf("error encoding json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing ImageResponse: %w", err)
	}

	return &result, nil
}
