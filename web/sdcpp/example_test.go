package sdcpp

import (
	"context"
	"fmt"
	"os"
)

func Example() {
	// Create a client (use Default for convenience with command-line flags)
	client := Default

	// Generate an image from a text prompt
	resp, err := client.Generate(context.Background(), ImageGenerationRequest{
		Prompt:       "a beautiful sunset over the ocean",
		N:            1,
		Size:         "512x512",
		OutputFormat: "png",
	})
	if err != nil {
		fmt.Println("Error generating image:", err)
		return
	}

	// Convert the first image to WebP
	webpBytes, err := resp.ToWebP(0, 95)
	if err != nil {
		fmt.Println("Error converting to WebP:", err)
		return
	}

	// Write to file
	if err := os.WriteFile("sunset.webp", webpBytes, 0644); err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	fmt.Println("Image saved as sunset.webp")
}

func Example_generateMultiple() {
	resp, err := Generate(context.Background(), ImageGenerationRequest{
		Prompt: "a cat sitting on a windowsill",
		N:      3,
		Size:   "768x512",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for i := range resp.Data {
		webp, err := resp.ToWebP(i, 90)
		if err != nil {
			fmt.Println("Error converting:", err)
			continue
		}
		fmt.Printf("Image %d converted to WebP (%d bytes)\n", i, len(webp))
	}
}

func Example_editImage() {
	// Load an image file
	imageData, err := os.ReadFile("input.png")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	resp, err := Edit(context.Background(), ImageEditRequest{
		Prompt: "add some clouds to the sky",
		Image:  [][]byte{imageData},
		N:      1,
	})
	if err != nil {
		fmt.Println("Error editing image:", err)
		return
	}

	avif, err := resp.ToAVIF(0, 85, 6)
	if err != nil {
		fmt.Println("Error converting to AVIF:", err)
		return
	}

	fmt.Printf("Edited image converted to AVIF (%d bytes)\n", len(avif))
}

func Example_listModels() {
	models, err := ListModels(context.Background())
	if err != nil {
		fmt.Println("Error listing models:", err)
		return
	}

	for _, model := range models.Data {
		fmt.Printf("Model: %s (owned by: %s)\n", model.ID, model.OwnedBy)
	}
}

func Example_customClient() {
	// Create a client with a custom server URL
	client := &Client{
		HTTP:      Default.HTTP,
		APIServer: "http://localhost:8080",
	}

	resp, err := client.Generate(context.Background(), ImageGenerationRequest{
		Prompt: "cyberpunk city at night",
		N:      1,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Generated %d images\n", len(resp.Data))
}

func Example_convertToMultipleFormats() {
	resp, err := Generate(context.Background(), ImageGenerationRequest{
		Prompt: "a serene mountain landscape",
		N:      1,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Convert to WebP
	webp, _ := resp.ToWebP(0, 95)
	fmt.Printf("WebP: %d bytes\n", len(webp))

	// Convert to AVIF
	avif, _ := resp.ToAVIF(0, 90, 6)
	fmt.Printf("AVIF: %d bytes\n", len(avif))
}
