package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"github.com/google/go-github/v81/github"
)

// GitHub limits webhook payloads to 25 MB.
const maxPayloadBytes = 25 * 1024 * 1024

// eventHandlers is the extension point for future event processing.
// Return an error to respond with HTTP 500. Handlers run synchronously and must
// tolerate repeated delivery of the same event. The logger includes delivery_id.
type eventHandlers interface {
	issueOpened(context.Context, *slog.Logger, *github.IssuesEvent) error
	issueCommentCreated(context.Context, *slog.Logger, *github.IssueCommentEvent) error
	pullRequestOpened(context.Context, *slog.Logger, *github.PullRequestEvent) error
	pullRequestCommentCreated(context.Context, *slog.Logger, *github.IssueCommentEvent) error
	pullRequestReviewCommentCreated(context.Context, *slog.Logger, *github.PullRequestReviewCommentEvent) error
	discussionCreated(context.Context, *slog.Logger, *github.DiscussionEvent) error
	discussionCommentCreated(context.Context, *slog.Logger, *github.DiscussionCommentEvent) error
}

type receiver struct {
	secret   []byte
	handlers eventHandlers
}

func (s *receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	lg := slog.Default().With("delivery_id", github.DeliveryID(r), "event", github.WebHookType(r))
	if len(s.secret) == 0 {
		lg.ErrorContext(r.Context(), "webhook secret is missing")
		http.Error(w, "webhook receiver is not configured", http.StatusServiceUnavailable)
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		http.Error(w, "use application/json", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadBytes)
	defer r.Body.Close()
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "payload is too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "cannot read payload", http.StatusBadRequest)
		}
		return
	}
	signature := r.Header.Get(github.SHA256SignatureHeader)
	if !strings.HasPrefix(signature, "sha256=") || github.ValidateSignature(signature, payload, s.secret) != nil {
		http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
		return
	}

	eventType := github.WebHookType(r)
	switch eventType {
	case "issues", "issue_comment", "pull_request", "pull_request_review_comment", "discussion", "discussion_comment", "ping":
	default:
		w.WriteHeader(http.StatusNoContent)
		return
	}
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		lg.WarnContext(r.Context(), "cannot parse webhook", "err", err)
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}
	if err := s.dispatch(r.Context(), lg, event); err != nil {
		lg.ErrorContext(r.Context(), "webhook handler failed", "err", err)
		http.Error(w, "webhook handler failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *receiver) dispatch(ctx context.Context, lg *slog.Logger, event any) error {
	switch event := event.(type) {
	case *github.IssuesEvent:
		if event.GetAction() == "opened" {
			return s.handlers.issueOpened(ctx, lg, event)
		}
	case *github.IssueCommentEvent:
		if event.GetAction() == "created" {
			if event.GetIssue().GetPullRequestLinks() != nil {
				return s.handlers.pullRequestCommentCreated(ctx, lg, event)
			}
			return s.handlers.issueCommentCreated(ctx, lg, event)
		}
	case *github.PullRequestEvent:
		if event.GetAction() == "opened" {
			return s.handlers.pullRequestOpened(ctx, lg, event)
		}
	case *github.PullRequestReviewCommentEvent:
		if event.GetAction() == "created" {
			return s.handlers.pullRequestReviewCommentCreated(ctx, lg, event)
		}
	case *github.DiscussionEvent:
		if event.GetAction() == "created" {
			return s.handlers.discussionCreated(ctx, lg, event)
		}
	case *github.DiscussionCommentEvent:
		if event.GetAction() == "created" {
			return s.handlers.discussionCommentCreated(ctx, lg, event)
		}
	}
	return nil
}
