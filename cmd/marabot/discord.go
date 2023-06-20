package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"within.website/ln"
	"within.website/ln/opname"
)

func (mr *MaraRevolt) importDiscordData(ctx context.Context, db *sql.DB, dg *discordgo.Session) error {
	ctx = opname.With(ctx, "import-discord-data")
	channels, err := dg.GuildChannels(*furryholeDiscord)
	if err != nil {
		return err
	}

	for _, ch := range channels {
		if _, err := db.Exec("INSERT INTO discord_channels (id, guild_id, name, topic, nsfw) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, topic = EXCLUDED.topic, nsfw = EXCLUDED.nsfw", ch.ID, ch.GuildID, ch.Name, ch.Topic, ch.NSFW); err != nil {
			ln.Error(ctx, err, ln.F{"channel_name": ch.Name, "channel_id": ch.ID})
			continue
		}
	}

	roles, err := dg.GuildRoles(*furryholeDiscord)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if _, err := db.ExecContext(ctx, "INSERT INTO discord_roles (guild_id, id, name, color, hoist, position) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, color = EXCLUDED.color, position = EXCLUDED.position", furryholeDiscord, role.ID, role.Name, fmt.Sprintf("#%06x", role.Color), role.Hoist, role.Position); err != nil {
			ln.Error(ctx, err, ln.Action("inserting role"))
			continue
		}
	}

	// https://cdn.discordapp.com/emojis/664686615616290816.webp?size=240&quality=lossless
	emoji, err := dg.GuildEmojis(*furryholeDiscord)
	if err != nil {
		return err
	}
	for _, emoji := range emoji {
		eURL := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s?size=240&quality=lossless", emoji.ID)
		if _, err := db.ExecContext(ctx, "INSERT INTO discord_emoji (id, guild_id, name, url) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url", emoji.ID, furryholeDiscord, emoji.Name, eURL); err != nil {
			ln.Error(ctx, err, ln.Action("inserting emoji"))
			continue
		}
		mr.attachmentPreprocess.Add([3]string{eURL, "emoji", ""}, len(eURL))
	}

	rows, err := mr.db.QueryContext(ctx, "SELECT url, message_id FROM discord_attachments WHERE url NOT IN ( SELECT url FROM s3_uploads )")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var url, messageID string
			if err := rows.Scan(&url, &messageID); err != nil {
				continue
			}

			mr.attachmentPreprocess.Add([3]string{url, "attachments", messageID}, len(url))
		}
	}

	return nil
}

func (mr *MaraRevolt) DiscordMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	ctx := opname.With(context.Background(), "marabot.discord-message-delete")
	if _, err := mr.db.ExecContext(ctx, "DELETE FROM discord_messages WHERE id = ?", m.ID); err != nil {
		ln.Error(ctx, err, ln.Action("nuking deleted messages"))
	}

	rows, err := mr.db.QueryContext(ctx, "SELECT id FROM s3_uploads WHERE message_id = ?", m.ID)
	if err != nil {
		ln.Error(ctx, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			ln.Error(ctx, err)
			return
		}

		if _, err := mr.db.ExecContext(ctx, "DELETE FROM s3_uploads WHERE id = ?", id); err != nil {
			ln.Error(ctx, err)
		}

		if _, err := mr.s3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
			Key: aws.String(m.ID),
		}); err != nil {
			ln.Error(ctx, err)
		}
	}
}

func (mr *MaraRevolt) DiscordMessageEdit(s *discordgo.Session, m *discordgo.MessageUpdate) {
	if _, err := mr.db.Exec("UPDATE discord_messages SET content = ?, edited_at = ? WHERE id = ?", m.Content, time.Now().Format(time.RFC3339), m.ID); err != nil {
		ln.Error(context.Background(), err)
	}
}

func (mr *MaraRevolt) DiscordMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if err := mr.discordMessageCreate(s, m); err != nil {
		ln.Error(context.Background(), err)
	}
}

func (mr *MaraRevolt) discordMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if _, err := mr.db.Exec(`INSERT INTO discord_users (id, username, avatar_url, accent_color)
VALUES (?, ?, ?, ?)
ON CONFLICT(id)
DO UPDATE SET username = EXCLUDED.username, avatar_url = EXCLUDED.avatar_url, accent_color = EXCLUDED.accent_color`, m.Author.ID, m.Author.Username, m.Author.AvatarURL(""), m.Author.AccentColor); err != nil {
		return err
	}

	mr.attachmentPreprocess.Add([3]string{m.Author.AvatarURL(""), "avatars", ""}, len(m.Author.Avatar))

	if _, err := mr.db.Exec(`INSERT INTO discord_messages (id, guild_id, channel_id, author_id, content, created_at, edited_at, webhook_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, m.ID, m.GuildID, m.ChannelID, m.Author.ID, m.Content, m.Timestamp.Format(time.RFC3339), m.EditedTimestamp, m.WebhookID); err != nil {
		return err
	}

	if m.WebhookID != "" {
		if _, err := mr.db.Exec("INSERT INTO discord_webhook_message_info (id, name, avatar_url) VALUES (?, ?, ?)", m.ID, m.Author.Username, m.Author.AvatarURL("")); err != nil {
			return err
		}
	}

	for _, att := range m.Attachments {
		if _, err := mr.db.Exec(`INSERT INTO discord_attachments (id, message_id, url, proxy_url, filename, content_type, width, height, size) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, att.ID, m.ID, att.URL, att.ProxyURL, att.Filename, att.ContentType, att.Width, att.Height, att.Size); err != nil {
			return err
		}

		mr.attachmentPreprocess.Add([3]string{att.URL, "attachments", m.ID}, len(att.URL))
	}

	for _, emb := range m.Embeds {
		if emb.Image == nil {
			continue
		}
		if _, err := mr.db.Exec(`INSERT INTO discord_attachments (id, message_id, url, proxy_url, filename, content_type, width, height, size) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), m.ID, emb.Image.URL, emb.Image.ProxyURL, filepath.Base(emb.Image.URL), "", emb.Image.Width, emb.Image.Height, 0); err != nil {
			return err
		}

		mr.attachmentPreprocess.Add([3]string{emb.Image.URL, "attachments", m.ID}, len(emb.Image.URL))
	}

	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		return err
	}

	if _, err := mr.db.Exec("INSERT INTO discord_channels (id, guild_id, name, topic, nsfw) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, topic = EXCLUDED.topic, nsfw = EXCLUDED.nsfw", ch.ID, ch.GuildID, ch.Name, ch.Topic, ch.NSFW); err != nil {
		return err
	}

	for _, emoji := range m.GetCustomEmojis() {
		eURL := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s?size=240&quality=lossless", emoji.ID)
		if _, err := mr.db.Exec("INSERT INTO discord_emoji (id, guild_id, name, url) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url", emoji.ID, furryholeDiscord, emoji.Name, eURL); err != nil {
			return err
		}
		mr.attachmentPreprocess.Add([3]string{eURL, "emoji", ""}, len(eURL))
	}

	return nil
}
