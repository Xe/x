package main

import (
	"context"
	"log/slog"

	"github.com/google/go-github/v81/github"
)

type loggingHandlers struct{}

func (loggingHandlers) issueOpened(ctx context.Context, lg *slog.Logger, event *github.IssuesEvent) error {
	lg.InfoContext(ctx, "issue opened", "repository", event.GetRepo().GetFullName(), "url", event.GetIssue().GetHTMLURL())
	return nil
}

func (loggingHandlers) issueCommentCreated(ctx context.Context, lg *slog.Logger, event *github.IssueCommentEvent) error {
	lg.InfoContext(ctx, "issue comment created", "repository", event.GetRepo().GetFullName(), "url", event.GetComment().GetHTMLURL())
	return nil
}

func (loggingHandlers) pullRequestOpened(ctx context.Context, lg *slog.Logger, event *github.PullRequestEvent) error {
	lg.InfoContext(ctx, "pull request opened", "repository", event.GetRepo().GetFullName(), "url", event.GetPullRequest().GetHTMLURL())
	return nil
}

func (loggingHandlers) pullRequestCommentCreated(ctx context.Context, lg *slog.Logger, event *github.IssueCommentEvent) error {
	lg.InfoContext(ctx, "pull request comment created", "repository", event.GetRepo().GetFullName(), "url", event.GetComment().GetHTMLURL())
	return nil
}

func (loggingHandlers) pullRequestReviewCommentCreated(ctx context.Context, lg *slog.Logger, event *github.PullRequestReviewCommentEvent) error {
	lg.InfoContext(ctx, "pull request review comment created", "repository", event.GetRepo().GetFullName(), "url", event.GetComment().GetHTMLURL())
	return nil
}

func (loggingHandlers) discussionCreated(ctx context.Context, lg *slog.Logger, event *github.DiscussionEvent) error {
	lg.InfoContext(ctx, "discussion created", "repository", event.GetRepo().GetFullName(), "url", event.GetDiscussion().GetHTMLURL())
	return nil
}

func (loggingHandlers) discussionCommentCreated(ctx context.Context, lg *slog.Logger, event *github.DiscussionCommentEvent) error {
	lg.InfoContext(ctx, "discussion comment created", "repository", event.GetRepo().GetFullName(), "url", event.GetComment().GetHTMLURL())
	return nil
}
