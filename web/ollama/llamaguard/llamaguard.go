package llamaguard

import (
	"context"
	"fmt"
	"strings"

	"within.website/x/web/ollama"
)

type Category string

const (
	S1  Category = "Violent Crimes"
	S2  Category = "Non-Violent Crimes"
	S3  Category = "Sex Crimes"
	S4  Category = "Child Exploitation"
	S5  Category = "Defamation"
	S6  Category = "Specialized Advice"
	S7  Category = "Privacy"
	S8  Category = "Intellectual Property"
	S9  Category = "Indiscriminate Weapons"
	S10 Category = "Hate"
	S11 Category = "Self-Harm"
	S12 Category = "Sexual Content"
	S13 Category = "Elections"
	S14 Category = "Code Interpreter Abuse"

	Unknown Category = "Unknown"
)

func (c Category) String() string {
	return string(c)
}

func ParseCategory(s string) Category {
	switch s {
	case "S1":
		return S1
	case "S2":
		return S2
	case "S3":
		return S3
	case "S4":
		return S4
	case "S5":
		return S5
	case "S6":
		return S6
	case "S7":
		return S7
	case "S8":
		return S8
	case "S9":
		return S9
	case "S10":
		return S10
	case "S11":
		return S11
	case "S12":
		return S12
	case "S13":
		return S13
	case "S14":
		return S14
	default:
		return ""
	}
}

type Response struct {
	IsSafe              bool       `json:"is_safe"`
	ViolationCategories []Category `json:"violation_categories"`
}

func formatMessages(messages []ollama.Message) string {
	var sb strings.Builder

	for _, m := range messages {
		switch m.Role {
		case "user":
			sb.WriteString("User: ")
		case "assistant":
			sb.WriteString("Agent: ")
		}
		sb.WriteString(m.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func Check(ctx context.Context, cli *ollama.Client, role, model string, messages []ollama.Message) (*Response, error) {
	req := &ollama.GenerateRequest{
		Model:     model,
		System:    &role,
		Prompt:    formatMessages(messages),
		KeepAlive: "9999m",
	}

	resp, err := cli.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llamaguard: failed to generate response: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("llamaguard: response was nil")
	}

	var result Response

	resp.Response = strings.TrimSpace(resp.Response)
	if resp.Response == "safe" {
		result.IsSafe = true
		return &result, nil
	}

	result.IsSafe = false

	reasons := strings.SplitN(resp.Response, "\n", 2)
	if len(reasons) != 2 {
		return nil, fmt.Errorf("llamaguard: response was not in the expected format")
	}

	for _, r := range strings.Split(reasons[1], ",") {
		result.ViolationCategories = append(result.ViolationCategories, ParseCategory(r))
	}

	return &result, nil
}
