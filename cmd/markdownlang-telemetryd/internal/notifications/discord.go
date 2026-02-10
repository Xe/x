// Package notifications handles sending notifications to Discord webhooks.
package notifications

import (
	"fmt"
	"net/http"
	"os"

	"within.website/x/web/discordwebhook"
)

const (
	// envDiscordWebhook is the environment variable for the new user notifications webhook
	envDiscordWebhook = "MARKDOWNLANG_TELEMETRY_DISCORD_WEBHOOK"

	// envXeWebhook is the environment variable for the quota exceeded notifications webhook
	envXeWebhook = "MARKDOWNLANG_TELEMETRY_XE_WEBHOOK"

	// quotaLimit is the number of executions after which Xe should be notified
	quotaLimit = 69
)

var (
	// discordWebhookURL is the webhook URL for new user notifications
	discordWebhookURL = os.Getenv(envDiscordWebhook)

	// xeWebhookURL is the webhook URL for quota exceeded notifications
	xeWebhookURL = os.Getenv(envXeWebhook)
)

// NotifyNewUser sends a Discord webhook notification when a new user is detected.
func NotifyNewUser(name, email string) error {
	if discordWebhookURL == "" {
		return nil // No webhook configured, silently skip
	}

	wh := discordwebhook.Webhook{
		Content:  fmt.Sprintf("New markdownlang user detected: **%s** (%s)", name, email),
		Username: "MarkdownLang Telemetry",
		Embeds: []discordwebhook.Embeds{
			{
				Title: "New User",
				Fields: []discordwebhook.EmbedField{
					{
						Name:   "Name",
						Value:  name,
						Inline: true,
					},
					{
						Name:   "Email",
						Value:  email,
						Inline: true,
					},
				},
			},
		},
		AllowedMentions: map[string][]string{
			"parse": {},
		},
	}

	req := discordwebhook.Send(discordWebhookURL, wh)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send new user notification: %w", err)
	}
	defer resp.Body.Close()

	if err := discordwebhook.Validate(resp); err != nil {
		return fmt.Errorf("failed to validate new user notification response: %w", err)
	}

	return nil
}

// NotifyXeForQuota sends a Discord webhook notification to Xe when a user exceeds the quota limit.
// The message includes a corporate sales message.
func NotifyXeForQuota(name, email string, count int) error {
	if xeWebhookURL == "" {
		return nil // No webhook configured, silently skip
	}

	wh := discordwebhook.Webhook{
		Content:  fmt.Sprintf("<@72838115944828928> User **%s** (%s) has executed markdownlang %d times!", name, email, count),
		Username: "MarkdownLang Telemetry",
		Embeds: []discordwebhook.Embeds{
			{
				Title:       "Quota Exceeded",
				Description: fmt.Sprintf("This user has exceeded the free tier limit of %d executions.", quotaLimit),
				Fields: []discordwebhook.EmbedField{
					{
						Name:   "Name",
						Value:  name,
						Inline: true,
					},
					{
						Name:   "Email",
						Value:  email,
						Inline: true,
					},
					{
						Name:   "Execution Count",
						Value:  fmt.Sprintf("%d", count),
						Inline: true,
					},
				},
				Footer: &discordwebhook.EmbedFooter{
					Text: "🏢 Corporate sales opportunity! Consider reaching out about enterprise licensing.",
				},
			},
		},
		AllowedMentions: map[string][]string{
			"parse": {"users"},
		},
	}

	req := discordwebhook.Send(xeWebhookURL, wh)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send quota exceeded notification: %w", err)
	}
	defer resp.Body.Close()

	if err := discordwebhook.Validate(resp); err != nil {
		return fmt.Errorf("failed to validate quota exceeded notification response: %w", err)
	}

	return nil
}
