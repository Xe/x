// Package sdcpp provides a client for the stable-diffusion.cpp HTTP server.
//
// See https://github.com/leejet/stable-diffusion.cpp for more information.
package sdcpp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/gen2brain/avif"
	"github.com/gen2brain/webp"
	"within.website/x/web"
)

var (
	sdServerURL = flag.String("within.website/x/web/sdcpp/server-url", "http://localhost:1234", "URL for the stable-diffusion.cpp API used with the default client")
)

func buildURL(base, path string) (*url.URL, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}

	u.Path = path

	return u, nil
}

// ModelData represents a single model in the models list response.
type ModelData struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse is the response from GET /v1/models.
type ModelsResponse struct {
	Data []ModelData `json:"data"`
}

// ImageGenerationRequest is a request to generate images from a text prompt.
type ImageGenerationRequest struct {
	Prompt            string `json:"prompt"`
	N                 int    `json:"n,omitempty"`                 // Number of images to generate (1-8, default 1)
	Size              string `json:"size,omitempty"`              // Size in format "WxH" (default "512x512")
	OutputFormat      string `json:"output_format,omitempty"`     // "png" or "jpeg" (default "png")
	OutputCompression int    `json:"output_compression,omitempty"` // For JPEG, 0-100 (default 100)
}

// ImageData represents a single generated image.
type ImageData struct {
	B64JSON string `json:"b64_json"`
}

// ImageGenerationResponse is the response from POST /v1/images/generations.
type ImageGenerationResponse struct {
	Created      string      `json:"created"`
	Data         []ImageData `json:"data"`
	OutputFormat string      `json:"output_format"`
}

// ImageEditRequest is a request to edit/modify images with a prompt.
type ImageEditRequest struct {
	Prompt            string   `json:"prompt"`
	Image             [][]byte `json:"-"`                    // Raw image data, not sent as JSON
	Mask              []byte   `json:"-"`                    // Optional mask data, not sent as JSON
	N                 int      `json:"n,omitempty"`          // Number of images to generate (1-8, default 1)
	Size              string   `json:"size,omitempty"`       // Size in format "WxH" (default "512x512")
	OutputFormat      string   `json:"output_format,omitempty"`     // "png" or "jpeg" (default "png")
	OutputCompression int      `json:"output_compression,omitempty"` // For JPEG, 0-100 (default 100)
}

// ImageEditResponse is the response from POST /v1/images/edits.
type ImageEditResponse = ImageGenerationResponse

var (
	Default *Client = &Client{
		HTTP:      http.DefaultClient,
		APIServer: *sdServerURL,
	}
)

// Generate sends an image generation request using the default client.
func Generate(ctx context.Context, req ImageGenerationRequest) (*ImageGenerationResponse, error) {
	return Default.Generate(ctx, req)
}

// ListModels lists available models using the default client.
func ListModels(ctx context.Context) (*ModelsResponse, error) {
	return Default.ListModels(ctx)
}

// Edit sends an image edit request using the default client.
func Edit(ctx context.Context, req ImageEditRequest) (*ImageEditResponse, error) {
	return Default.Edit(ctx, req)
}

type Client struct {
	HTTP      *http.Client
	APIServer string
}

// ListModels returns the list of available models from the server.
func (c *Client) ListModels(ctx context.Context) (*ModelsResponse, error) {
	u, err := buildURL(c.APIServer, "/v1/models")
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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

	var result ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing ModelsResponse: %w", err)
	}

	return &result, nil
}

// Generate generates images from a text prompt.
func (c *Client) Generate(ctx context.Context, req ImageGenerationRequest) (*ImageGenerationResponse, error) {
	u, err := buildURL(c.APIServer, "/v1/images/generations")
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return nil, fmt.Errorf("error encoding json: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error fetching response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result ImageGenerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing ImageGenerationResponse: %w", err)
	}

	return &result, nil
}

// Edit edits/modifies images with a prompt.
func (c *Client) Edit(ctx context.Context, req ImageEditRequest) (*ImageEditResponse, error) {
	u, err := buildURL(c.APIServer, "/v1/images/edits")
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add prompt field
	if err := writer.WriteField("prompt", req.Prompt); err != nil {
		return nil, fmt.Errorf("error writing prompt field: %w", err)
	}

	// Add optional fields
	if req.N != 0 {
		if err := writer.WriteField("n", fmt.Sprintf("%d", req.N)); err != nil {
			return nil, fmt.Errorf("error writing n field: %w", err)
		}
	}
	if req.Size != "" {
		if err := writer.WriteField("size", req.Size); err != nil {
			return nil, fmt.Errorf("error writing size field: %w", err)
		}
	}
	if req.OutputFormat != "" {
		if err := writer.WriteField("output_format", req.OutputFormat); err != nil {
			return nil, fmt.Errorf("error writing output_format field: %w", err)
		}
	}
	if req.OutputCompression != 0 {
		if err := writer.WriteField("output_compression", fmt.Sprintf("%d", req.OutputCompression)); err != nil {
			return nil, fmt.Errorf("error writing output_compression field: %w", err)
		}
	}

	// Add images
	for i, img := range req.Image {
		part, err := writer.CreateFormFile("image[]", fmt.Sprintf("image%d", i))
		if err != nil {
			return nil, fmt.Errorf("error creating form file for image: %w", err)
		}
		if _, err := io.Copy(part, bytes.NewReader(img)); err != nil {
			return nil, fmt.Errorf("error writing image data: %w", err)
		}
	}

	// Add optional mask
	if len(req.Mask) > 0 {
		part, err := writer.CreateFormFile("mask", "mask")
		if err != nil {
			return nil, fmt.Errorf("error creating form file for mask: %w", err)
		}
		if _, err := io.Copy(part, bytes.NewReader(req.Mask)); err != nil {
			return nil, fmt.Errorf("error writing mask data: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("error closing multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &body)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error fetching response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result ImageEditResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing ImageEditResponse: %w", err)
	}

	return &result, nil
}

// DecodeImage decodes the base64-encoded image data from ImageData into an image.Image.
func (r *ImageGenerationResponse) DecodeImage(idx int) (image.Image, string, error) {
	if idx < 0 || idx >= len(r.Data) {
		return nil, "", fmt.Errorf("index %d out of range [0, %d)", idx, len(r.Data))
	}

	imgBytes, err := base64.StdEncoding.DecodeString(r.Data[idx].B64JSON)
	if err != nil {
		return nil, "", fmt.Errorf("error decoding base64: %w", err)
	}

	img, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, "", fmt.Errorf("error decoding image: %w", err)
	}

	return img, format, nil
}

// ToWebP converts the image at index idx to WebP format with the specified quality.
// Quality should be between 0 and 100, where higher values mean better quality.
func (r *ImageGenerationResponse) ToWebP(idx int, quality int) ([]byte, error) {
	img, _, err := r.DecodeImage(idx)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("error encoding webp: %w", err)
	}

	return buf.Bytes(), nil
}

// ToAVIF converts the image at index idx to AVIF format with the specified quality and speed.
// Quality should be between 0 and 100, where higher values mean better quality.
// Speed should be between 0 (slowest, best compression) and 10 (fastest, worst compression).
func (r *ImageGenerationResponse) ToAVIF(idx int, quality int, speed int) ([]byte, error) {
	img, _, err := r.DecodeImage(idx)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := avif.Encode(&buf, img, avif.Options{Quality: quality, Speed: speed}); err != nil {
		return nil, fmt.Errorf("error encoding avif: %w", err)
	}

	return buf.Bytes(), nil
}
