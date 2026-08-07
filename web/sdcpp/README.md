# sdcpp

Package sdcpp provides a Go client for the [stable-diffusion.cpp](https://github.com/leejet/stable-diffusion.cpp) HTTP server.

stable-diffusion.cpp is a lightweight, pure C/C++ implementation of Stable Diffusion that runs locally on CPU. This client allows you to generate images from text prompts, edit existing images (img2img), and convert results to modern image formats like WebP and AVIF.

## Installation

```go
import "within.website/x/web/sdcpp"
```

## Server Setup

First, run the stable-diffusion.cpp server:

```bash
# Clone and build
git clone https://github.com/leejet/stable-diffusion.cpp
cd stable-diffusion.cpp
mkdir build && cd build
cmake ..
cmake --build . --config Release

# Run the server (default: http://localhost:1234)
./bin/server --model-path /path/to/model.safetensors
```

## Configuration

The client uses a command-line flag for the server URL:

```bash
-go flag="within.website/x/web/sdcpp/server-url=http://localhost:1234"
```

## Basic Usage

### Generate Images

```go
import (
    "context"
    "fmt"
    "os"
    "within.website/x/web/sdcpp"
)

func main() {
    // Use the default client
    resp, err := sdcpp.Generate(context.Background(), sdcpp.ImageGenerationRequest{
        Prompt:       "a beautiful sunset over the ocean",
        N:            1,
        Size:         "512x512",
        OutputFormat: "png",
    })
    if err != nil {
        panic(err)
    }

    // Convert to WebP
    webpBytes, err := resp.ToWebP(0, 95)
    if err != nil {
        panic(err)
    }

    os.WriteFile("sunset.webp", webpBytes, 0644)
}
```

### Custom Client

```go
client := &sdcpp.Client{
    HTTP:      http.DefaultClient,
    APIServer: "http://localhost:8080",
}

resp, err := client.Generate(ctx, sdcpp.ImageGenerationRequest{
    Prompt: "cyberpunk city at night",
    N:      1,
})
```

### Edit Images (img2img)

```go
imageData, _ := os.ReadFile("input.png")

resp, err := sdcpp.Edit(context.Background(), sdcpp.ImageEditRequest{
    Prompt: "add some clouds to the sky",
    Image:  [][]byte{imageData},
    N:      1,
})

// Convert to AVIF
avifBytes, _ := resp.ToAVIF(0, 85, 6)
os.WriteFile("output.avif", avifBytes, 0644)
```

### Generate Multiple Images

```go
resp, err := sdcpp.Generate(ctx, sdcpp.ImageGenerationRequest{
    Prompt: "a cat sitting on a windowsill",
    N:      3,
    Size:   "768x512",
})

for i := range resp.Data {
    webp, _ := resp.ToWebP(i, 90)
    os.WriteFile(fmt.Sprintf("cat_%d.webp", i), webp, 0644)
}
```

### List Models

```go
models, err := sdcpp.ListModels(ctx)
for _, model := range models.Data {
    fmt.Printf("Model: %s\n", model.ID)
}
```

## Image Conversion

The response object provides methods to convert images to modern formats:

| Method | Description |
|--------|-------------|
| `resp.DecodeImage(idx)` | Decode to `image.Image` |
| `resp.ToWebP(idx, quality)` | Convert to WebP (quality: 0-100) |
| `resp.ToAVIF(idx, quality, speed)` | Convert to AVIF (quality: 0-100, speed: 0-10) |

## Request Parameters

### ImageGenerationRequest

| Field | Type | Description |
|-------|------|-------------|
| `Prompt` | string | Text description of desired image |
| `N` | int | Number of images to generate (1-8) |
| `Size` | string | Image size in "WxH" format (default "512x512") |
| `OutputFormat` | string | "png" or "jpeg" (default "png") |
| `OutputCompression` | int | JPEG quality 0-100 (default 100) |

### ImageEditRequest

| Field | Type | Description |
|-------|------|-------------|
| `Prompt` | string | Text description of desired changes |
| `Image` | [][]byte | Input image(s) to edit |
| `Mask` | []byte | Optional mask for selective editing |
| `N` | int | Number of images to generate (1-8) |
| `Size` | string | Image size in "WxH" format |
| `OutputFormat` | string | "png" or "jpeg" |

## API Compatibility

This client implements the OpenAI Images API compatible endpoints from stable-diffusion.cpp:

- `GET /v1/models` - List available models
- `POST /v1/images/generations` - Generate images (text-to-image)
- `POST /v1/images/edits` - Edit images (img2img)
