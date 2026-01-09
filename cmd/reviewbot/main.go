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
	"os"
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

	ai := openai.NewClient(
		option.WithAPIKey(*openAIAPIKey),
		option.WithBaseURL(*openAIAPIBase),
	)
	_ = ai

	gh := github.NewClient(http.DefaultClient).WithAuthToken(*githubToken)

	repo, err := gitfs.NewRepo("https://github.com/" + *githubRepo + ".git")
	if err != nil {
		log.Fatalf("can't initialize repo: %v", err)
	}

	hash, fs, err := repo.Clone(*githubSha)
	if err != nil {
		log.Fatalf("can't clone %s commit %s: %v", *githubRepo, *githubSha, err)
	}
	_ = fs

	slog.Info("cloned repo filesystem", "commit-hash", hash.String())

	details := strings.SplitN(*githubRepo, "/", 2)
	owner, repoName := details[0], details[1]

	pr, resp, err := gh.PullRequests.Get(ctx, owner, repoName, *prNumber)
	if err != nil {
		log.Fatalf("can't get PR information: status %d: %v", resp.StatusCode, err)
	}
	slog.Info("got PR info", "title", pr.Title, "node-id", pr.NodeID)

	files, resp, err := gh.PullRequests.ListFiles(ctx, owner, repoName, *prNumber, nil)
	if err != nil {
		log.Fatalf("can't list files for PR: status %d: %v", resp.StatusCode, err)
	}

	commits, resp, err := gh.PullRequests.ListCommits(ctx, owner, repoName, *prNumber, nil)
	if err != nil {
		log.Fatalf("can't list files for PR: status %d: %v", resp.StatusCode, err)
	}

	tmpl, err := template.New("ai-prompt").Parse(promptTemplate)
	if err != nil {
		log.Fatalf("can't parse prompt: %v", err)
	}

	var agentsMD string
	if fin, err := fs.Open("AGENTS.md"); err == nil {
		data, err := io.ReadAll(fin)
		if err != nil {
			log.Fatalf("can't read AGENTS.md: %v", err)
		}

		agentsMD = string(data)
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(io.MultiWriter(&buf, os.Stdout), struct {
		Files      []*github.CommitFile
		Commits    []*github.RepositoryCommit
		Title      string
		Author     string
		AuthorRole string
		AgentsMD   string
		PRBody     string
	}{
		Files:      files,
		Commits:    commits,
		Title:      *pr.Title,
		Author:     *pr.User.Login,
		AuthorRole: *pr.AuthorAssociation,
		AgentsMD:   agentsMD,
		PRBody:     *pr.Body,
	}); err != nil {
		log.Fatal(err)
	}

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
			log.Fatalf("can't get chat completion: %v", err)
		}

		msg := completion.Choices[0].Message
		toolCalls := msg.ToolCalls
		if len(toolCalls) == 0 {
			fmt.Println(msg.Content)
			break
		}

		params.Messages = append(params.Messages, msg.ToParam())

		for _, toolCall := range toolCalls {
			switch toolCall.Function.Name {
			case "submit_review":
				var args SubmitReviewParams
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					log.Fatalf("can't unmarshal tool call args: %s yielded %v", toolCall.Function.Arguments, err)
				}

				if err := args.Valid(); err != nil {
					log.Fatalf("can't validate tool call args: %s yielded %v", toolCall.Function.Arguments, err)
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
					log.Fatalf("can't create PR comment: status %d: %v", resp.StatusCode, err)
				}
				fmt.Println("Created review:", comment.HTMLURL)
				finalize = true
			default:
				log.Fatalf("model invoked unknown tool %s with args %s", toolCall.Function.Name, toolCall.Function.Arguments)
			}
		}
	}
}
