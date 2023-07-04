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
	"github.com/jackc/pgx/v5"
	"within.website/ln"
	"within.website/ln/opname"
)

func (mr *MaraRevolt) importDiscordData(ctx context.Context, db *sql.DB, dg *discordgo.Session) error {
	ctx = opname.With(ctx, "import-discord-data")

	tx, err := mr.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	channels, err := dg.GuildChannels(*furryholeDiscord)
	if err != nil {
		return err
	}

	for _, ch := range channels {
		if _, err := tx.Exec(ctx, "INSERT INTO discord_channels (id, guild_id, name, topic, nsfw) VALUES ($1, $2, $3, $4, $5) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, topic = EXCLUDED.topic, nsfw = EXCLUDED.nsfw", ch.ID, ch.GuildID, ch.Name, ch.Topic, ch.NSFW); err != nil {
			ln.Error(ctx, err, ln.F{"channel_name": ch.Name, "channel_id": ch.ID})
			continue
		}
	}

	roles, err := dg.GuildRoles(*furryholeDiscord)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if _, err := tx.Exec(ctx, "INSERT INTO discord_roles (guild_id, id, name, color, hoist, position) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, color = EXCLUDED.color, position = EXCLUDED.position", furryholeDiscord, role.ID, role.Name, fmt.Sprintf("#%06x", role.Color), role.Hoist, role.Position); err != nil {
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
		if _, err := tx.Exec(ctx, "INSERT INTO discord_emoji (id, guild_id, name, url) VALUES ($1, $2, $3, $4) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url", emoji.ID, furryholeDiscord, emoji.Name, eURL); err != nil {
			ln.Error(ctx, err, ln.Action("inserting emoji"))
			continue
		}

		if err := mr.archiveAttachment(ctx, tx, eURL, "emoji", ""); err != nil {
			return err
		}
	}

	rows, err := tx.Query(ctx, "SELECT url, message_id FROM discord_attachments WHERE url NOT IN ( SELECT url FROM s3_uploads )")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var url, messageID string
			if err := rows.Scan(&url, &messageID); err != nil {
				continue
			}

			if err := mr.archiveAttachment(ctx, tx, url, "attachments", messageID); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (mr *MaraRevolt) DiscordMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	ctx := opname.With(context.Background(), "marabot.discord-message-delete")

	tx, err := mr.pg.Begin(ctx)
	if err != nil {
		ln.Error(ctx, err)
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "DELETE FROM discord_messages WHERE id = $1", m.ID); err != nil {
		ln.Error(ctx, err, ln.Action("nuking deleted messages"))
	}

	rows, err := tx.Query(ctx, "SELECT id FROM s3_uploads WHERE message_id = $1", m.ID)
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

		if _, err := tx.Exec(ctx, "DELETE FROM s3_uploads WHERE id = ?", id); err != nil {
			ln.Error(ctx, err)
		}

		if _, err := mr.s3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
			Key:    aws.String(m.ID),
			Bucket: awsS3Bucket,
		}); err != nil {
			ln.Error(ctx, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		ln.Error(ctx, err)
		return
	}
}

func (mr *MaraRevolt) DiscordMessageEdit(s *discordgo.Session, m *discordgo.MessageUpdate) {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	if _, err := mr.pg.Exec(context.Background(), "UPDATE discord_messages SET content = ?, edited_at = ? WHERE id = ?", m.Content, time.Now().Format(time.RFC3339), m.ID); err != nil {
		ln.Error(context.Background(), err)
	}
}

func (mr *MaraRevolt) DiscordMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	mr.lock.Lock()
	defer mr.lock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = opname.With(ctx, "marabot.discordMessageCreate")

	tx, err := mr.pg.Begin(ctx)
	if err != nil {
		ln.Error(ctx, err)
		return
	}
	defer tx.Rollback(ctx)

	if err := mr.discordMessageCreate(ctx, tx, s, m.Message); err != nil {
		ln.Error(ctx, err, ln.F{
			"channel_id": m.ChannelID,
			"message_id": m.ID,
		})
		s.MessageReactionAdd(m.ChannelID, m.ID, "🔥")
	}

	if err := tx.Commit(ctx); err != nil {
		ln.Error(ctx, err, ln.F{
			"channel_id": m.ChannelID,
			"message_id": m.ID,
		})
	}
}

func (mr *MaraRevolt) DiscordReactionAdd(s *discordgo.Session, mra *discordgo.MessageReactionAdd) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = opname.With(ctx, "marabot.discord-reaction-add")

	if mra.Emoji.Name == "💾" {
		go mr.backfillDiscordChannel(s, mra.ChannelID, mra.MessageID)
		return
	}

	m, err := s.ChannelMessage(mra.ChannelID, mra.MessageID)
	if err != nil {
		ln.Error(ctx, err)
		return
	}
	defer s.MessageReactionRemove(m.ChannelID, m.ID, "🔥", "@me")

	tx, err := mr.pg.Begin(ctx)
	if err != nil {
		ln.Error(ctx, err)
		return
	}
	defer tx.Rollback(ctx)

	if err := mr.discordMessageCreate(ctx, tx, s, m); err != nil {
		ln.Error(context.Background(), err, ln.F{
			"channel_id": m.ChannelID,
			"message_id": m.ID,
		})
		s.MessageReactionAdd(m.ChannelID, m.ID, "😭")
	}

	if err := tx.Commit(ctx); err != nil {
		ln.Error(ctx, err, ln.F{
			"channel_id": m.ChannelID,
			"message_id": m.ID,
		})
	}
}

func (mr *MaraRevolt) doesDiscordMessageExist(ctx context.Context, tx pgx.Tx, messageID string) (bool, error) {
	var count int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM discord_messages WHERE id = ?", messageID).Scan(&count); err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	return false, nil
}

func (mr *MaraRevolt) backfillDiscordChannel(s *discordgo.Session, channelID, messageID string) {
	curr := messageID
	ctx := opname.With(context.Background(), "marabot.backfillDiscordChannel")

	ln.Log(ctx, ln.Action("archiving channel from message"), ln.F{"channel_id": channelID, "message_id": messageID})

	tx, err := mr.pg.Begin(ctx)
	if err != nil {
		ln.Error(ctx, err)
		return
	}
	defer tx.Rollback(ctx)

	t := time.NewTicker(30 * time.Second)
	defer t.Stop()

	done := false

	for range t.C {
		ln.Log(ctx, ln.Action("fetching batch of messages"), ln.F{"curr": curr})
		msgs, err := s.ChannelMessages(channelID, 100, "", curr, "")
		if err != nil {
			ln.Error(ctx, err)
			s.ChannelMessageSend(channelID, fmt.Sprintf("error getting messages past %s: %v", curr, err))
			break
		}

		for _, msg := range msgs {
			found, err := mr.doesDiscordMessageExist(ctx, tx, msg.ID)
			if err != nil {
				ln.Error(ctx, err, ln.F{"message_id": msg.ID})
				continue
			}

			if found {
				ln.Log(ctx, ln.Action("stopping archival"))
				done = true
			}

			if err := mr.discordMessageCreate(ctx, tx, s, msg); err != nil {
				ln.Error(ctx, err, ln.F{"message_id": msg.ID})
				continue
			}

			curr = msg.ID
		}

		if done {
			break
		}
	}

	if err := tx.Commit(ctx); err != nil {
		ln.Error(ctx, err, ln.F{
			"channel_id": channelID,
			"message_id": messageID,
		})
	}
}

func (mr *MaraRevolt) discordMessageCreate(ctx context.Context, tx pgx.Tx, s *discordgo.Session, m *discordgo.Message) error {
	if _, err := tx.Exec(ctx, `INSERT INTO discord_users (id, username, avatar_url, accent_color)
VALUES ($1, $2, $3, $4)
ON CONFLICT(id)
DO UPDATE SET username = EXCLUDED.username, avatar_url = EXCLUDED.avatar_url, accent_color = EXCLUDED.accent_color`, m.Author.ID, m.Author.Username, m.Author.AvatarURL(""), m.Author.AccentColor); err != nil {
		return err
	}

	if err := mr.archiveAttachment(ctx, tx, m.Author.AvatarURL(""), "avatars", ""); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO discord_messages (id, guild_id, channel_id, author_id, content, created_at, edited_at, webhook_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT(id) DO UPDATE SET content = EXCLUDED.content`, m.ID, m.GuildID, m.ChannelID, m.Author.ID, m.Content, m.Timestamp.Format(time.RFC3339), m.EditedTimestamp, m.WebhookID); err != nil {
		return err
	}

	if m.WebhookID != "" {
		if _, err := tx.Exec(ctx, "INSERT INTO discord_webhook_message_info (id, name, avatar_url) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", m.ID, m.Author.Username, m.Author.AvatarURL("")); err != nil {
			return err
		}
	}

	for _, att := range m.Attachments {
		if _, err := tx.Exec(ctx, `INSERT INTO discord_attachments (id, message_id, url, proxy_url, filename, content_type, width, height, size) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT DO NOTHING`, att.ID, m.ID, att.URL, att.ProxyURL, att.Filename, att.ContentType, att.Width, att.Height, att.Size); err != nil {
			return err
		}

		if err := mr.archiveAttachment(ctx, tx, att.URL, "attachments", m.ID); err != nil {
			return err
		}

		for _, emb := range m.Embeds {
			if emb.Image == nil {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO discord_attachments (id, message_id, url, proxy_url, filename, content_type, width, height, size) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT DO NOTHING`, uuid.NewString(), m.ID, emb.Image.URL, emb.Image.ProxyURL, filepath.Base(emb.Image.URL), "", emb.Image.Width, emb.Image.Height, 0); err != nil {
				return err
			}

			if err := mr.archiveAttachment(ctx, tx, emb.Image.URL, "attachments", m.ID); err != nil {
				return err
			}
		}

		ch, err := s.Channel(m.ChannelID)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, "INSERT INTO discord_channels (id, guild_id, name, topic, nsfw) VALUES ($1, $2, $3, $4, $5) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, topic = EXCLUDED.topic, nsfw = EXCLUDED.nsfw", ch.ID, ch.GuildID, ch.Name, ch.Topic, ch.NSFW); err != nil {
			return err
		}

		for _, emoji := range m.GetCustomEmojis() {
			eURL := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s?size=240&quality=lossless", emoji.ID)
			if _, err := tx.Exec(ctx, "INSERT INTO discord_emoji (id, guild_id, name, url) VALUES ($1, $2, $3, $4) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url", emoji.ID, furryholeDiscord, emoji.Name, eURL); err != nil {
				return err
			}
			mr.attachmentPreprocess.Add([3]string{eURL, "emoji", ""}, len(eURL))
			if err := mr.archiveAttachment(ctx, tx, eURL, "emoji", m.ID); err != nil {
				return err
			}
		}
	}

	return nil
}
