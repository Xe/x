package dalle

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"within.website/x/web"
)

type Client struct {
	apiKey string
}

func New(apiKey string) Client {
	return Client{apiKey: apiKey}
}

type Model string

const (
	DALLE2 = Model("dall-e-2")
	DALLE3 = Model("dall-e-3")
)

type Size string

const (
	Size256  = Size("256x256")
	Size512  = Size("512x512")
	Size1024 = Size("1024x1024")

	// These are only supported with the dall-e-3 model.
	SizeHDWide = Size("1792x1024")
	SizeHDTall = Size("1024x1792")
)

type Style string

const (
	StyleVivid   = Style("vivid")
	StyleNatural = Style("natural")
)

type Options struct {
	Model   Model   `json:"model"`
	Prompt  string  `json:"prompt"`
	N       *int    `json:"n,omitempty"`
	Quality *string `json:"quality,omitempty"`
	Size    *Size   `json:"size"`
	Style   *Style  `json:"style,omitempty"`
	User    *string `json:"user,omitempty"`
}

type Image struct {
	URL string `json:"url"`
}

func (i Image) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("url", i.URL),
	)
}

type Response struct {
	Created int     `json:"created"`
	Data    []Image `json:"data"`
}

func (r Response) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("created", r.Created),
		slog.Any("data", r.Data),
	)
}

func (c Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	return http.DefaultClient.Do(req)
}

func (c Client) GenerateImage(ctx context.Context, opts Options) (*Response, error) {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(opts); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/images/generations", buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
