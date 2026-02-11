package ai

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"sort"

	"github.com/google/subcommands"
	"github.com/rodaine/table"
)

type ListModelsCmd struct {
	provider string
}

func (*ListModelsCmd) Name() string     { return "ai-list-models" }
func (*ListModelsCmd) Synopsis() string { return "List models available from an AI provider" }
func (*ListModelsCmd) Usage() string {
	return `ai-list-models --provider <name>
  Query the /v1/models endpoint of a configured AI provider and list available models.
`
}

func (c *ListModelsCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.provider, "provider", "", "Provider name to query")
}

func (c *ListModelsCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if c.provider == "" {
		fmt.Println("--provider is required")
		fmt.Println(c.Usage())
		return subcommands.ExitUsageError
	}

	providers, err := loadProviders()
	if err != nil {
		fmt.Printf("error loading providers: %v\n", err)
		return subcommands.ExitFailure
	}

	var found *Provider
	for _, p := range providers {
		if p.Name == c.provider {
			found = &p
			break
		}
	}

	if found == nil {
		fmt.Printf("provider %q not found; add it with ai-add-provider first\n", c.provider)
		return subcommands.ExitFailure
	}

	models, err := fetchModels(ctx, found)
	if err != nil {
		fmt.Printf("error fetching models: %v\n", err)
		return subcommands.ExitFailure
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	tbl := table.New("ID", "Owned By")
	for _, m := range models {
		tbl.AddRow(m.ID, m.OwnedBy)
	}
	tbl.Print()

	return subcommands.ExitSuccess
}

type model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func fetchModels(ctx context.Context, p *Provider) ([]model, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var result struct {
		Data []model `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("can't decode response: %w", err)
	}

	return result.Data, nil
}
