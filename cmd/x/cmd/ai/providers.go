package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Provider holds the configuration for a single AI API provider.
type Provider struct {
	Name    string `json:"name"`
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("can't find config dir: %w", err)
	}

	dir = filepath.Join(dir, "within.website", "x")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("can't create config dir: %w", err)
	}

	return filepath.Join(dir, "ai-providers.json"), nil
}

func loadProviders() ([]Provider, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("can't read %s: %w", path, err)
	}

	var providers []Provider
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, fmt.Errorf("can't parse %s: %w", path, err)
	}

	return providers, nil
}

func saveProviders(providers []Provider) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return fmt.Errorf("can't marshal providers: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("can't write %s: %w", path, err)
	}

	return nil
}
