package sdcpp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeImageGenerationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Fake base64 encoded PNG (1x1 red pixel PNG)
		fakePNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="

		resp := ImageGenerationResponse{
			Created:      "2024-01-01T00:00:00Z",
			OutputFormat: "png",
			Data: []ImageData{
				{B64JSON: fakePNG},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func fakeModelsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp := ModelsResponse{
			Data: []ModelData{
				{
					ID:      "sd-cpp-local",
					Object:  "model",
					OwnedBy: "local",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func fakeImageEditHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Fake base64 encoded PNG
		fakePNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="

		resp := ImageEditResponse{
			Created:      "2024-01-01T00:00:00Z",
			OutputFormat: "png",
			Data: []ImageData{
				{B64JSON: fakePNG},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func TestGenerate(t *testing.T) {
	srv := httptest.NewServer(fakeImageGenerationHandler())
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIServer: srv.URL}

	req := ImageGenerationRequest{
		Prompt:       "a beautiful sunset",
		N:            1,
		Size:         "512x512",
		OutputFormat: "png",
	}

	resp, err := c.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Created == "" {
		t.Fatal("expected Created timestamp, got empty string")
	}

	if resp.OutputFormat != "png" {
		t.Fatalf("expected output format 'png', got '%s'", resp.OutputFormat)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(resp.Data))
	}

	if resp.Data[0].B64JSON == "" {
		t.Fatal("expected base64 encoded image, got empty string")
	}

	// Verify it's valid base64
	_, err = base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
	if err != nil {
		t.Fatalf("expected valid base64 string, got error: %v", err)
	}
}

func TestGenerateWithPackageFunction(t *testing.T) {
	srv := httptest.NewServer(fakeImageGenerationHandler())
	defer srv.Close()

	// Save and restore the default client's APIServer
	originalURL := Default.APIServer
	defer func() { Default.APIServer = originalURL }()
	Default.APIServer = srv.URL

	req := ImageGenerationRequest{
		Prompt: "a cat sitting on a table",
		N:      1,
	}

	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(resp.Data))
	}
}

func TestGenerateMultipleImages(t *testing.T) {
	fakePNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp := ImageGenerationResponse{
			Created:      "2024-01-01T00:00:00Z",
			OutputFormat: "png",
			Data: []ImageData{
				{B64JSON: fakePNG},
				{B64JSON: fakePNG},
				{B64JSON: fakePNG},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIServer: srv.URL}

	req := ImageGenerationRequest{
		Prompt: "generate 3 images",
		N:      3,
	}

	resp, err := c.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 images, got %d", len(resp.Data))
	}
}

func TestGenerateJPEGFormat(t *testing.T) {
	fakeJPEG := "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAP//////////////////////////////////////wAALCAACAgBAREA/8QAFAABAAAAAAAAAAAAAAAAAAAACv/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAD8AT//Z"

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp := ImageGenerationResponse{
			Created:      "2024-01-01T00:00:00Z",
			OutputFormat: "jpeg",
			Data: []ImageData{
				{B64JSON: fakeJPEG},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIServer: srv.URL}

	req := ImageGenerationRequest{
		Prompt:            "a sunset",
		OutputFormat:      "jpeg",
		OutputCompression: 90,
	}

	resp, err := c.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.OutputFormat != "jpeg" {
		t.Fatalf("expected output format 'jpeg', got '%s'", resp.OutputFormat)
	}
}

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(fakeModelsHandler())
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIServer: srv.URL}

	resp, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 model, got %d", len(resp.Data))
	}

	if resp.Data[0].ID != "sd-cpp-local" {
		t.Fatalf("expected model ID 'sd-cpp-local', got '%s'", resp.Data[0].ID)
	}

	if resp.Data[0].Object != "model" {
		t.Fatalf("expected object type 'model', got '%s'", resp.Data[0].Object)
	}

	if resp.Data[0].OwnedBy != "local" {
		t.Fatalf("expected owned_by 'local', got '%s'", resp.Data[0].OwnedBy)
	}
}

func TestListModelsWithPackageFunction(t *testing.T) {
	srv := httptest.NewServer(fakeModelsHandler())
	defer srv.Close()

	// Save and restore the default client's APIServer
	originalURL := Default.APIServer
	defer func() { Default.APIServer = originalURL }()
	Default.APIServer = srv.URL

	resp, err := ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 model, got %d", len(resp.Data))
	}
}

func TestEdit(t *testing.T) {
	srv := httptest.NewServer(fakeImageEditHandler())
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIServer: srv.URL}

	// Create a simple fake image (1x1 PNG)
	fakeImageData, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==")

	req := ImageEditRequest{
		Prompt: "make it blue",
		Image:  [][]byte{fakeImageData},
		N:      1,
		Size:   "512x512",
	}

	resp, err := c.Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(resp.Data))
	}

	if resp.Data[0].B64JSON == "" {
		t.Fatal("expected base64 encoded image, got empty string")
	}
}

func TestEditWithPackageFunction(t *testing.T) {
	srv := httptest.NewServer(fakeImageEditHandler())
	defer srv.Close()

	// Save and restore the default client's APIServer
	originalURL := Default.APIServer
	defer func() { Default.APIServer = originalURL }()
	Default.APIServer = srv.URL

	fakeImageData, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==")

	req := ImageEditRequest{
		Prompt: "add some clouds",
		Image:  [][]byte{fakeImageData},
	}

	resp, err := Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(resp.Data))
	}
}

func TestEditWithMask(t *testing.T) {
	srv := httptest.NewServer(fakeImageEditHandler())
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIServer: srv.URL}

	fakeImageData, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==")

	req := ImageEditRequest{
		Prompt: "fill the masked area",
		Image:  [][]byte{fakeImageData},
		Mask:   fakeImageData,
	}

	resp, err := c.Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(resp.Data))
	}
}

func TestEditMultipleImages(t *testing.T) {
	srv := httptest.NewServer(fakeImageEditHandler())
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIServer: srv.URL}

	fakeImageData, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==")

	req := ImageEditRequest{
		Prompt: "variations",
		Image:  [][]byte{fakeImageData, fakeImageData},
	}

	resp, err := c.Edit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(resp.Data))
	}
}

func TestGenerateHTTPError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error": "prompt required",
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIServer: srv.URL}

	req := ImageGenerationRequest{
		Prompt: "",
	}

	_, err := c.Generate(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for bad request, got nil")
	}

	if !strings.Contains(err.Error(), "wanted status code 200") {
		t.Fatalf("expected status code error, got: %v", err)
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		path      string
		wantPath  string
		wantErr   bool
	}{
		{
			name:     "simple path",
			base:     "http://localhost:1234",
			path:     "/v1/models",
			wantPath: "/v1/models",
			wantErr:  false,
		},
		{
			name:     "base with path - replaced",
			base:     "http://localhost:1234/api",
			path:     "/v1/models",
			wantPath: "/v1/models",
			wantErr:  false,
		},
		{
			name:     "invalid base URL",
			base:     "://invalid",
			path:     "/v1/models",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := buildURL(tt.base, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && u.Path != tt.wantPath {
				t.Errorf("buildURL() path = %v, want %v", u.Path, tt.wantPath)
			}
		})
	}
}

func TestDecodeImage(t *testing.T) {
	fakePNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="

	resp := ImageGenerationResponse{
		Data: []ImageData{{B64JSON: fakePNG}},
	}

	img, format, err := resp.DecodeImage(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if img == nil {
		t.Fatal("expected image, got nil")
	}

	if format != "png" {
		t.Fatalf("expected format 'png', got '%s'", format)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 1 || bounds.Dy() != 1 {
		t.Fatalf("expected 1x1 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestDecodeImageInvalidIndex(t *testing.T) {
	resp := ImageGenerationResponse{
		Data: []ImageData{},
	}

	_, _, err := resp.DecodeImage(0)
	if err == nil {
		t.Fatal("expected error for invalid index, got nil")
	}
}

func TestDecodeImageInvalidBase64(t *testing.T) {
	resp := ImageGenerationResponse{
		Data: []ImageData{{B64JSON: "!!!invalid-base64!!!"}},
	}

	_, _, err := resp.DecodeImage(0)
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestToWebP(t *testing.T) {
	fakePNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="

	resp := ImageGenerationResponse{
		Data: []ImageData{{B64JSON: fakePNG}},
	}

	webpBytes, err := resp.ToWebP(0, 95)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(webpBytes) == 0 {
		t.Fatal("expected webp bytes, got empty slice")
	}

	// Verify it's a valid WebP by checking the RIFF header
	if len(webpBytes) < 12 {
		t.Fatal("webp data too short to be valid")
	}
	if string(webpBytes[0:4]) != "RIFF" {
		t.Fatalf("expected RIFF header, got %q", webpBytes[0:4])
	}
	if string(webpBytes[8:12]) != "WEBP" {
		t.Fatalf("expected WEBP signature, got %q", webpBytes[8:12])
	}
}

func TestToAVIF(t *testing.T) {
	fakePNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="

	resp := ImageGenerationResponse{
		Data: []ImageData{{B64JSON: fakePNG}},
	}

	avifBytes, err := resp.ToAVIF(0, 90, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(avifBytes) == 0 {
		t.Fatal("expected avif bytes, got empty slice")
	}

	// Verify it's a valid AVIF by checking the ftyp header
	// AVIF files start with "ftypavif" at byte 4
	if len(avifBytes) < 12 {
		t.Fatal("avif data too short to be valid")
	}
	if string(avifBytes[4:8]) != "ftyp" {
		t.Fatalf("expected ftyp header, got %q", avifBytes[4:8])
	}
}

