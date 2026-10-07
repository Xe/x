package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-github/v81/github"
)

type recordingHandlers struct {
	called []string
	event  any
	err    error
}

func (h *recordingHandlers) record(name string, event any) error {
	h.called = append(h.called, name)
	h.event = event
	return h.err
}

func (h *recordingHandlers) issueOpened(_ context.Context, _ *slog.Logger, event *github.IssuesEvent) error {
	return h.record("issueOpened", event)
}

func (h *recordingHandlers) issueCommentCreated(_ context.Context, _ *slog.Logger, event *github.IssueCommentEvent) error {
	return h.record("issueCommentCreated", event)
}

func (h *recordingHandlers) pullRequestOpened(_ context.Context, _ *slog.Logger, event *github.PullRequestEvent) error {
	return h.record("pullRequestOpened", event)
}

func (h *recordingHandlers) pullRequestCommentCreated(_ context.Context, _ *slog.Logger, event *github.IssueCommentEvent) error {
	return h.record("pullRequestCommentCreated", event)
}

func (h *recordingHandlers) pullRequestReviewCommentCreated(_ context.Context, _ *slog.Logger, event *github.PullRequestReviewCommentEvent) error {
	return h.record("pullRequestReviewCommentCreated", event)
}

func (h *recordingHandlers) discussionCreated(_ context.Context, _ *slog.Logger, event *github.DiscussionEvent) error {
	return h.record("discussionCreated", event)
}

func (h *recordingHandlers) discussionCommentCreated(_ context.Context, _ *slog.Logger, event *github.DiscussionCommentEvent) error {
	return h.record("discussionCommentCreated", event)
}

func signedRequest(event, body, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(github.EventTypeHeader, event)
	req.Header.Set(github.DeliveryIDHeader, "test-delivery")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	req.Header.Set(github.SHA256SignatureHeader, "sha256="+hex.EncodeToString(mac.Sum(nil)))
	return req
}

func TestWebhookRouting(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, event, body, want string
	}{
		{"issue", "issues", `{"action":"opened","issue":{"number":42}}`, "issueOpened"},
		{"issue comment", "issue_comment", `{"action":"created","issue":{"number":42},"comment":{"body":"hello"}}`, "issueCommentCreated"},
		{"PR", "pull_request", `{"action":"opened","pull_request":{"number":42}}`, "pullRequestOpened"},
		{"PR conversation comment", "issue_comment", `{"action":"created","issue":{"number":42,"pull_request":{}},"comment":{"body":"hello"}}`, "pullRequestCommentCreated"},
		{"PR review comment", "pull_request_review_comment", `{"action":"created","comment":{"body":"hello"}}`, "pullRequestReviewCommentCreated"},
		{"discussion", "discussion", `{"action":"created","discussion":{"number":42}}`, "discussionCreated"},
		{"discussion comment", "discussion_comment", `{"action":"created","comment":{"body":"hello"}}`, "discussionCommentCreated"},
		{"issue closed", "issues", `{"action":"closed"}`, ""},
		{"issue comment edited", "issue_comment", `{"action":"edited"}`, ""},
		{"PR reopened", "pull_request", `{"action":"reopened"}`, ""},
		{"PR review comment deleted", "pull_request_review_comment", `{"action":"deleted"}`, ""},
		{"discussion edited", "discussion", `{"action":"edited"}`, ""},
		{"discussion comment deleted", "discussion_comment", `{"action":"deleted"}`, ""},
		{"ping", "ping", `{"zen":"hello"}`, ""},
		{"unsupported event", "push", `{}`, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handlers := &recordingHandlers{}
			srv := &receiver{secret: []byte("secret"), handlers: handlers}
			resp := httptest.NewRecorder()
			srv.ServeHTTP(resp, signedRequest(tt.event, tt.body, "secret"))
			if resp.Code != http.StatusNoContent {
				t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
			}
			if tt.want == "" {
				if len(handlers.called) != 0 {
					t.Fatalf("unexpected handlers: %v", handlers.called)
				}
			} else if len(handlers.called) != 1 || handlers.called[0] != tt.want {
				t.Fatalf("handlers = %v, want [%s]", handlers.called, tt.want)
			}
			// The handler receives the full typed payload, including comment content.
			if event, ok := handlers.event.(*github.IssueCommentEvent); ok && event.GetComment().GetBody() != "hello" {
				t.Fatal("comment body was not preserved")
			}
		})
	}
}

func TestWebhookErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		change func(*http.Request, *receiver, *recordingHandlers)
		status int
		calls  int
	}{
		{"missing signature", func(r *http.Request, _ *receiver, _ *recordingHandlers) { r.Header.Del(github.SHA256SignatureHeader) }, 401, 0},
		{"wrong secret", func(_ *http.Request, s *receiver, _ *recordingHandlers) { s.secret = []byte("wrong") }, 401, 0},
		{"tampered body", func(r *http.Request, _ *receiver, _ *recordingHandlers) { r.Body = http.NoBody }, 401, 0},
		{"missing server secret", func(_ *http.Request, s *receiver, _ *recordingHandlers) { s.secret = nil }, 503, 0},
		{"wrong method", func(r *http.Request, _ *receiver, _ *recordingHandlers) { r.Method = http.MethodGet }, 405, 0},
		{"wrong content type", func(r *http.Request, _ *receiver, _ *recordingHandlers) { r.Header.Set("Content-Type", "text/plain") }, 415, 0},
		{"invalid JSON", func(r *http.Request, _ *receiver, _ *recordingHandlers) { *r = *signedRequest("issues", "{", "secret") }, 400, 0},
		{"invalid typed field", func(r *http.Request, _ *receiver, _ *recordingHandlers) {
			*r = *signedRequest("issues", `{"action":"opened","issue":{"number":"bad"}}`, "secret")
		}, 400, 0},
		{"oversized payload", func(r *http.Request, _ *receiver, _ *recordingHandlers) {
			r.Body = http.NoBody
			*r = *signedRequest("issues", strings.Repeat("x", maxPayloadBytes+1), "secret")
		}, 413, 0},
		{"handler failure", func(_ *http.Request, _ *receiver, h *recordingHandlers) { h.err = errors.New("processing failed") }, 500, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handlers := &recordingHandlers{}
			srv := &receiver{secret: []byte("secret"), handlers: handlers}
			req := signedRequest("issues", `{"action":"opened","issue":{"number":42}}`, "secret")
			tt.change(req, srv, handlers)
			resp := httptest.NewRecorder()
			srv.ServeHTTP(resp, req)
			if resp.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", resp.Code, tt.status, resp.Body.String())
			}
			if len(handlers.called) != tt.calls {
				t.Fatalf("handler calls = %d, want %d", len(handlers.called), tt.calls)
			}
		})
	}
}
