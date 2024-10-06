package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	flyapi "github.com/superfly/flyctl/api"
	"within.website/x/internal"
	"within.website/x/internal/yeet"
	"within.website/x/web/openai/chatgpt"
)

var (
	flyAPIBaseURL = flag.String("fly-base-url", "https://api.fly.io", "Fly API base URL")
	flyToken      = flag.String("fly-token", "", "Fly API token")
	openAIModel   = flag.String("openai-model", "gpt-3.5-turbo-16k", "OpenAI model to use")
	openAIToken   = flag.String("openai-token", "", "OpenAI API token")
	openAIURL     = flag.String("openai-url", "", "OpenAI API base URL")

	//go:embed playbooks/*
	playbooks embed.FS
)

func main() {
	internal.HandleStartup()

	if *openAIToken == "" {
		log.Fatal("must provide --openai-token")
	}

	if *flyToken == "" {
		log.Fatal("must provide --fly-token")
	}

	cli := chatgpt.NewClient(*openAIToken)

	if *openAIURL != "" {
		cli = cli.WithBaseURL(*openAIURL)
	}

	flyapi.SetBaseURL(*flyAPIBaseURL)
	fly := flyapi.NewClient(*flyToken, "alvis", "devel", flySlogger{})

	var playbook Playbook
	data, err := playbooks.ReadFile("playbooks/xe-pronouns-healthcheck.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(data, &playbook); err != nil {
		log.Fatal(err)
	}

	page := Page{
		Service: "xe-pronouns",
		Details: "health check failed for fly app xe-pronouns",
		ID:      "P0001",
	}

	incident := NewIncident(cli, fly, playbook, page)

	for {
		if err := incident.Step(); err != nil {
			log.Fatal(err)
		}

		if !incident.FinishedAt.IsZero() {
			break
		}

		if !incident.EscalatedAt.IsZero() {
			break
		}
	}

	if err := incident.Summarize(); err != nil {
		log.Fatal(err)
	}

	fout, err := os.Create(fmt.Sprintf("incidents/incident-%s.json", incident.Page.ID))
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	enc := json.NewEncoder(fout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(incident); err != nil {
		log.Fatal(err)
	}
}

type restartAppArgs struct {
	App    string `json:"app"`
	Reason string `json:"reason"`
}

func (r restartAppArgs) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("app", r.App),
		slog.String("reason", r.Reason),
	)
}

type waitArgs struct {
	DurationMinutes int `json:"duration"`
}

func (w waitArgs) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("duration", w.DurationMinutes),
	)
}

type closeArgs struct {
	Reason string `json:"reason"`
}

func (c closeArgs) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("reason", c.Reason),
	)
}

func runcmd(cmdName string, args ...string) (string, error) {
	ctx := context.Background()

	slog.Info("running command", "cmd", cmdName, "args", args)

	result, err := yeet.Output(ctx, cmdName, args...)
	if err != nil {
		return "", err
	}

	return result, nil
}

type Playbook struct {
	Meta struct {
		Service   string `json:"service"`
		Condition string `json:"condition"`
	} `json:"meta"`
	Details        string `json:"details"`
	HealthCheckURL string `json:"health_check_url"`
}

func (p Playbook) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("service", p.Meta.Service),
		slog.String("condition", p.Meta.Condition),
	)
}

type Page struct {
	Service string `json:"service"`
	Details string `json:"details"`
	ID      string `json:"id"`
}

func (p Page) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("service", p.Service),
		slog.String("details", p.Details),
		slog.String("id", p.ID),
	)
}

type Annotation struct {
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}

func (a Annotation) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Time("time", a.Time),
		slog.String("text", a.Text),
	)
}

type Incident struct {
	Playbook    Playbook          `json:"playbook"`
	Page        Page              `json:"page"`
	StartedAt   time.Time         `json:"started_at"`
	FinishedAt  time.Time         `json:"finished_at,omitempty"`
	EscalatedAt time.Time         `json:"escalated_at,omitempty"`
	Messages    []chatgpt.Message `json:"messages,omitempty"`
	Annotations []Annotation      `json:"annotations,omitempty"`
	CloseReason string            `json:"close_reason,omitempty"`
	Summary     string            `json:"summary,omitempty"`

	cli chatgpt.Client `json:"-"`
	fly *flyapi.Client `json:"-"`
}

func NewIncident(cli chatgpt.Client, fly *flyapi.Client, playbook Playbook, page Page) *Incident {
	messages := []chatgpt.Message{
		{
			Role:    "system",
			Content: basePrompt + "\n\n" + playbook.Details,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Error: %s", page.Details),
		},
	}

	return &Incident{
		Playbook:  playbook,
		Page:      page,
		StartedAt: time.Now(),
		Messages:  messages,
		cli:       cli,
		fly:       fly,
	}
}

func (i *Incident) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("service", i.Page.Service),
		slog.String("details", i.Page.Details),
		slog.String("id", i.Page.ID),
		slog.Time("started_at", i.StartedAt),
		slog.Int("message_count", len(i.Messages)),
	)
}

func (i *Incident) Step() error {
	ctx := context.Background()

	resp, err := i.cli.Complete(ctx, chatgpt.Request{
		Model:     *openAIModel,
		Messages:  i.Messages,
		Functions: functions,
	})
	if err != nil {
		return err
	}

	msg := resp.Choices[0].Message

	i.Messages = append(i.Messages, msg)

	slog.Info("got response", "message", msg.Content, "function", msg.FunctionCall)

	if resp.Choices[0].Message.FunctionCall != nil {
		if err := i.ExecFunction(ctx, msg); err != nil {
			return err
		}
	} else {
		slog.Info("incident note", "incident", i, "note", msg.Content)
	}

	return nil
}

func (i *Incident) Annotate(text string) {
	i.Annotations = append(i.Annotations, Annotation{
		Time: time.Now(),
		Text: text,
	})
}

func (i *Incident) Annotatef(format string, args ...interface{}) {
	i.Annotate(fmt.Sprintf(format, args...))
}

func (i *Incident) Reply(reason string) {
	i.Messages = append(i.Messages, chatgpt.Message{
		Role:    "user",
		Content: reason,
	})

	slog.Info("reply to incident", "incident", i, "reason", reason)
}

func (i *Incident) Replyf(format string, args ...interface{}) {
	i.Reply(fmt.Sprintf(format, args...))
}

func (i *Incident) Close(reason string) {
	i.FinishedAt = time.Now()
	slog.Info("closing incident", "incident", i, "reason", reason)
	i.Annotatef("closed incident: %s", reason)
	i.CloseReason = reason
}

func (i *Incident) Escalate() {
	slog.Error("escalating incident", "incident", i)
}

func (i *Incident) ExecFunction(ctx context.Context, msg chatgpt.Message) error {
	switch msg.FunctionCall.Name {
	case "restart_fly_app":
		var args restartAppArgs
		if err := json.Unmarshal([]byte(msg.FunctionCall.Arguments), &args); err != nil {
			return err
		}

		i.Annotatef("restarting app %s: %s", args.App, args.Reason)

		slog.Info("got restart_fly_app", "args", args)

		if _, err := i.fly.RestartApp(ctx, args.App); err != nil {
			i.Annotatef("error restarting app: %s", err)
			i.Replyf("error restarting app: %s", err)
			return nil
		}

		i.Replyf("restarted app %s successfully", args.App)
		i.Annotate("restarted app successfully")

		time.Sleep(5 * time.Second)

	case "perform_health_check":
		log.Println("performing health check")
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		if err := performFunctionFiveTimesWithDelay(ctx, i.Playbook.HealthCheckURL); err != nil {
			i.Replyf("health check failed:\n\n%s", err)
		} else {
			i.Reply("health check passed")
		}

	case "wait":
		var args waitArgs
		if err := json.Unmarshal([]byte(msg.FunctionCall.Arguments), &args); err != nil {
			return err
		}
		slog.Info("got wait", "args", args)

		i.Annotatef("waiting %d minutes", args.DurationMinutes)
		time.Sleep(time.Duration(args.DurationMinutes) * time.Minute)
		i.Annotate("done!")

	case "close":
		var args closeArgs
		if err := json.Unmarshal([]byte(msg.FunctionCall.Arguments), &args); err != nil {
			return err
		}
		slog.Info("got close", "args", args)
		i.Close(args.Reason)

	case "escalate":
		i.Escalate()
	}

	return nil
}

func (i *Incident) Summarize() error {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Incident %s\n", i.Page.ID)
	fmt.Fprintf(&sb, "Service: %s\n", i.Page.Service)
	fmt.Fprintf(&sb, "Details: %s\n", i.Page.Details)
	fmt.Fprintf(&sb, "Started at: %s\n", i.StartedAt)
	fmt.Fprintf(&sb, "Finished at: %s\n", i.FinishedAt)
	fmt.Fprintf(&sb, "Close reason: %s\n", i.CloseReason)

	for _, ann := range i.Annotations {
		fmt.Fprintf(&sb, "%s: %s\n", ann.Time.Format(time.Kitchen), ann.Text)
	}

	resp, err := i.cli.Complete(context.Background(), chatgpt.Request{
		Model: *openAIModel,
		Messages: []chatgpt.Message{
			{
				Role:    "system",
				Content: summaryPrompt,
			},
			{
				Role:    "user",
				Content: sb.String(),
			},
		},
	})
	if err != nil {
		return err
	}

	i.Summary = resp.Choices[0].Message.Content

	return nil
}

func performFunctionFiveTimesWithDelay(ctx context.Context, url string) error {
	passCount := 0
	var errs []error
	for i := 0; i < 5; i++ {
		if err := healthCheck(ctx, url); err == nil {
			passCount++
		} else {
			slog.Error("error performing function", "url", url, "err", err)
			errs = append(errs, err)
		}
		time.Sleep(time.Second)
	}

	if passCount >= 3 {
		return errors.Join(errs...)
	} else {
		return nil
	}
}

func healthCheck(ctx context.Context, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status %s", resp.Status)
	}

	return nil
}
