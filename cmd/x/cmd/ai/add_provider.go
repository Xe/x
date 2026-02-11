package ai

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

type AddProviderCmd struct {
	name    string
	apiKey  string
	baseURL string
}

func (*AddProviderCmd) Name() string     { return "ai-add-provider" }
func (*AddProviderCmd) Synopsis() string { return "Add or update an AI provider configuration" }
func (*AddProviderCmd) Usage() string {
	return `ai-add-provider --name <name> --base-url <url> [--api-key <key>]
  Add or update an AI API provider entry in ~/.config/within.website/x/ai-providers.json.
`
}

func (c *AddProviderCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.name, "name", "", "Provider name (e.g. openai, groq, ollama)")
	f.StringVar(&c.apiKey, "api-key", "", "API key for the provider")
	f.StringVar(&c.baseURL, "base-url", "", "Base URL for the provider API")
}

func (c *AddProviderCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if c.name == "" || c.baseURL == "" {
		fmt.Println("--name and --base-url are required")
		fmt.Println(c.Usage())
		return subcommands.ExitUsageError
	}

	providers, err := loadProviders()
	if err != nil {
		fmt.Printf("error loading providers: %v\n", err)
		return subcommands.ExitFailure
	}

	updated := false
	for i, p := range providers {
		if p.Name == c.name {
			providers[i].APIKey = c.apiKey
			providers[i].BaseURL = c.baseURL
			updated = true
			break
		}
	}

	if !updated {
		providers = append(providers, Provider{
			Name:    c.name,
			APIKey:  c.apiKey,
			BaseURL: c.baseURL,
		})
	}

	if err := saveProviders(providers); err != nil {
		fmt.Printf("error saving providers: %v\n", err)
		return subcommands.ExitFailure
	}

	if updated {
		fmt.Printf("updated provider %q\n", c.name)
	} else {
		fmt.Printf("added provider %q\n", c.name)
	}

	return subcommands.ExitSuccess
}
