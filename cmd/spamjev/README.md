# spamjev

spamjev receives GitHub webhooks at `POST /webhook`. The initial handlers log
event metadata. They do not change GitHub resources.

## Run

Set `WEBHOOK_SECRET` to the secret for the GitHub webhook. Then run:

```sh
go run ./cmd/spamjev --bind :8080
```

The default bind address is `:8080`. Configuration uses the standard repository
startup helpers. The `--webhook-secret` flag overrides `WEBHOOK_SECRET`.

## GitHub configuration

Set the payload URL to `https://YOUR_HOST/webhook`. Select `application/json`
as the content type. Set the same secret as `WEBHOOK_SECRET`.

Subscribe to these events:

| Event                         | Action    | Handler                                              |
| ----------------------------- | --------- | ---------------------------------------------------- |
| `issues`                      | `opened`  | `issueOpened`                                        |
| `issue_comment`               | `created` | `issueCommentCreated` or `pullRequestCommentCreated` |
| `pull_request`                | `opened`  | `pullRequestOpened`                                  |
| `pull_request_review_comment` | `created` | `pullRequestReviewCommentCreated`                    |
| `discussion`                  | `created` | `discussionCreated`                                  |
| `discussion_comment`          | `created` | `discussionCommentCreated`                           |

GitHub sends PR conversation comments as `issue_comment` events. The receiver
uses `issue.pull_request` to select the PR handler. Inline review comments use
`pull_request_review_comment` events.

## Extend

Add processing to the typed stubs in `handlers.go`, or implement `eventHandlers`.
Each handler receives the request context, a logger, and the full typed payload.
The logger includes the GitHub delivery ID and event type.

Handlers run synchronously. They must tolerate duplicate deliveries and finish
within the server timeout. The framework does not store or deduplicate events.

## Responses

The receiver requires an HMAC-SHA256 signature and limits payloads to 25 MiB.
Successful deliveries, pings, unsupported events, and ignored actions receive
`204 No Content`. Handler failures receive `500 Internal Server Error`.
GitHub does not automatically redeliver failed webhooks.
See [Handling failed webhook deliveries](https://docs.github.com/en/webhooks/using-webhooks/handling-failed-webhook-deliveries).

Invalid signatures receive `401`. Invalid JSON receives `400`. Unsupported
content types receive `415`. Oversized payloads receive `413`.

## Tests

```sh
go test ./cmd/spamjev
```
