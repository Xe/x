package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"text/template"

	"github.com/google/go-github/v81/github"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"rsc.io/gitfs"
	"within.website/x/internal"
)

var (
	githubRepo    = flag.String("github-repo", "", "GitHub repository")
	githubSha     = flag.String("github-sha", "", "GitHub commit SHA")
	githubToken   = flag.String("github-token", "", "GitHub API token")
	openAIAPIBase = flag.String("openai-api-base", "", "OpenAI API base URL")
	openAIAPIKey  = flag.String("openai-api-key", "", "OpenAI API key")
	openAIModel   = flag.String("openai-model", "gpt-oss-120b", "OpenAI model")
	prNumber      = flag.Int("pr-number", 0, "Pull request number")

	//go:embed prompt.tmpl
	promptTemplate string

	//go:embed systemprompt.txt
	systemPrompt string
)

func run(ctx context.Context) error {
	gh := github.NewClient(http.DefaultClient).WithAuthToken(*githubToken)

	repo, err := gitfs.NewRepo("https://github.com/" + *githubRepo + ".git")
	if err != nil {
		return fmt.Errorf("initializing repo: %w", err)
	}

	hash, fs, err := repo.Clone(*githubSha)
	if err != nil {
		return fmt.Errorf("cloning repo %s at commit %s: %w", *githubRepo, *githubSha, err)
	}

	slog.Info("cloned repo filesystem", "commit-hash", hash.String())

	details := strings.SplitN(*githubRepo, "/", 2)
	owner, repoName := details[0], details[1]

	pr, resp, err := gh.PullRequests.Get(ctx, owner, repoName, *prNumber)
	if err != nil {
		return fmt.Errorf("getting PR information: status %d: %w", resp.StatusCode, err)
	}
	slog.Info("got PR info", "title", pr.Title, "node-id", pr.NodeID)

	files, resp, err := gh.PullRequests.ListFiles(ctx, owner, repoName, *prNumber, nil)
	if err != nil {
		return fmt.Errorf("listing files for PR: status %d: %w", resp.StatusCode, err)
	}

	commits, resp, err := gh.PullRequests.ListCommits(ctx, owner, repoName, *prNumber, nil)
	if err != nil {
		return fmt.Errorf("listing commits for PR: status %d: %w", resp.StatusCode, err)
	}

	tmpl, err := template.New("ai-prompt").Parse(promptTemplate)
	if err != nil {
		return fmt.Errorf("parsing prompt template: %w", err)
	}

	var agentsMD string
	if fin, err := fs.Open("AGENTS.md"); err == nil {
		data, err := io.ReadAll(fin)
		if err != nil {
			return fmt.Errorf("reading AGENTS.md: %w", err)
		}

		agentsMD = string(data)
	}

	var buf bytes.Buffer

	prTitle := github.Stringify(pr.Title)
	prBody := github.Stringify(pr.Body)
	var authorLogin string
	if pr.User != nil && pr.User.Login != nil {
		authorLogin = *pr.User.Login
	}

	if err := tmpl.Execute(&buf, struct {
		Files    []*github.CommitFile
		Commits  []*github.RepositoryCommit
		Title    string
		Author   string
		AgentsMD string
		PRBody   string
	}{
		Files:    files,
		Commits:  commits,
		Title:    prTitle,
		Author:   authorLogin,
		AgentsMD: agentsMD,
		PRBody:   prBody,
	}); err != nil {
		return fmt.Errorf("executing prompt template: %w", err)
	}

	ai := openai.NewClient(
		option.WithAPIKey(*openAIAPIKey),
		option.WithBaseURL(*openAIAPIBase),
	)

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(buf.String()),
		},
		Tools: []openai.ChatCompletionToolUnionParam{
			// {
			// 	OfFunction: &openai.ChatCompletionFunctionToolParam{
			// 		Function: openai.FunctionDefinitionParam{
			// 			Name:        "python",
			// 			Description: openai.String("Execute python code"),
			// 			Parameters: openai.FunctionParameters{
			// 				"type": "object",
			// 				"properties": map[string]any{
			// 					"code": map[string]any{
			// 						"type": "string",
			// 					},
			// 				},
			// 				"required": []string{"code"},
			// 			},
			// 		},
			// 	},
			// },
			{
				OfFunction: &openai.ChatCompletionFunctionToolParam{
					Function: openai.FunctionDefinitionParam{
						Name:        "submit_review",
						Description: openai.String("Submit final pull request review"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"approved": map[string]any{
									"type": "boolean",
								},
								"message": map[string]any{
									"type": "string",
								},
							},
							"required": []string{"approved", "message"},
						},
					},
				},
			},
		},
	}

	finalize := false
	for !finalize {
		completion, err := ai.Chat.Completions.New(ctx, params)
		if err != nil {
			return fmt.Errorf("getting chat completion: %w", err)
		}

		msg := completion.Choices[0].Message
		toolCalls := msg.ToolCalls
		if len(toolCalls) == 0 {
			fmt.Println(msg.Content)
			event := "COMMENT"
			comment, resp, err := gh.PullRequests.CreateReview(ctx, owner, repoName, *prNumber, &github.PullRequestReviewRequest{
				Body:     &msg.Content,
				Event:    &event,
				CommitID: githubSha,
			})
			if err != nil {
				return fmt.Errorf("creating PR review: status %d: %w", resp.StatusCode, err)
			}
			fmt.Println("Created review:", comment.HTMLURL)
			break
		}

		params.Messages = append(params.Messages, msg.ToParam())

		for _, toolCall := range toolCalls {
			switch toolCall.Function.Name {
			case "submit_review":
				var args SubmitReviewParams
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					return fmt.Errorf("unmarshaling tool call args %s: %w", toolCall.Function.Arguments, err)
				}

				if err := args.Valid(); err != nil {
					return fmt.Errorf("validating tool call args %s: %w", toolCall.Function.Arguments, err)
				}

				slog.Info("got tool call", "name", toolCall.Function.Name, "args", args)

				event := "REQUEST_CHANGES"
				if args.Approved {
					event = "APPROVE"
				}

				comment, resp, err := gh.PullRequests.CreateReview(ctx, owner, repoName, *prNumber, &github.PullRequestReviewRequest{
					Body:     &args.Message,
					Event:    &event,
					CommitID: githubSha,
				})
				if err != nil {
					return fmt.Errorf("creating PR review: status %d: %w", resp.StatusCode, err)
				}
				fmt.Println("Created review:", comment.HTMLURL)
				finalize = true
			default:
				return fmt.Errorf("model invoked unknown tool %s with args %s", toolCall.Function.Name, toolCall.Function.Arguments)
			}
		}
	}

	return nil
}

func main() {
	internal.HandleStartup()

	slog.Info(
		"Starting up",
		"github-repo", *githubRepo,
		"github-sha", *githubSha,
		"has-github-token", *githubToken != "",
		"openai-api-base", *openAIAPIBase,
		"has-openai-api-key", *openAIAPIKey != "",
		"openai-model", *openAIModel,
		"pr-number", *prNumber,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalf("reviewbot run failed: %v", err)
	}
}
