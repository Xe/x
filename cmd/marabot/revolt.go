package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"within.website/ln"
	"within.website/x/internal"
	"within.website/x/internal/bundler"
	"within.website/x/web/revolt"
)

type MaraRevolt struct {
	cli                  *revolt.Client
	db                   *sql.DB
	ircmsgs              chan string
	attachmentPreprocess *bundler.Bundler[[3]string]
	attachmentUpload     *bundler.Bundler[*Attachment]
	uploader             *s3manager.Uploader
	s3                   s3iface.S3API

	revolt.NullHandler
}

func (mr *MaraRevolt) MessageCreate(ctx context.Context, msg *revolt.Message) error {
	if msg.Content == "!ping" {
		sendMsg := &revolt.SendMessage{
			Masquerade: &revolt.Masquerade{
				Name:      "cipra",
				AvatarURL: "https://cdn.xeiaso.net/avatar/" + internal.Hash("cipra", "yasomi"),
			},
		}
		sendMsg.SetContent("🏓 Pong!")

		if _, err := mr.cli.MessageReply(ctx, msg.ChannelId, msg.ID, true, sendMsg); err != nil {
			return err
		}
	}

	if err := msg.CalculateCreationDate(); err != nil {
		return err
	}

	if _, err := mr.db.ExecContext(ctx, "INSERT INTO revolt_messages(id, channel_id, author_id, content, created_at) VALUES (?, ?, ?, ?, ?)", msg.ID, msg.ChannelId, msg.AuthorId, msg.Content, msg.CreatedAt.Format(time.RFC3339)); err != nil {
		ln.Error(ctx, err, ln.Action("saving revolt message to database"))
	}

	if msg.Masquerade != nil {
		if _, err := mr.db.ExecContext(ctx, "INSERT INTO revolt_message_masquerade(id, username, avatar_url) VALUES(?, ?, ?)", msg.ID, msg.Masquerade.Name, msg.Masquerade.AvatarURL); err != nil {
			ln.Error(ctx, err, ln.Action("saving masquerade info to database"))
		}

		mr.attachmentPreprocess.Add([3]string{msg.Masquerade.AvatarURL, "avatars", ""}, len(msg.Masquerade.AvatarURL))
	}

	author, err := mr.cli.FetchUser(ctx, msg.AuthorId)
	if err != nil {
		return err
	}
	if err := author.CalculateCreationDate(); err != nil {
		return err
	}
	avatarURL := mr.cli.ResolveAttachment(author.Avatar)

	if _, err := mr.db.ExecContext(ctx, "INSERT INTO revolt_users(id, username, avatar_url, created_at) VALUES(?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET username = EXCLUDED.username, avatar_url = EXCLUDED.avatar_url, created_at = EXCLUDED.created_at", author.Id, author.Username, avatarURL, author.CreatedAt.Format(time.RFC3339)); err != nil {
		ln.Error(ctx, err, ln.Action("writing revolt user record"))
	}

	for _, att := range msg.Attachments {
		url := mr.cli.ResolveAttachment(att)

		if _, err := mr.db.ExecContext(ctx, "INSERT INTO revolt_attachments(id, tag, message_id, url, filename, content_type, width, height, size) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", att.ID, att.Tag, msg.ID, url, att.FileName, att.ContentType, att.Metadata.Width, att.Metadata.Height, att.Size); err != nil {
			ln.Error(ctx, err, ln.Action("writing revolt attachment information"))
		}

		mr.attachmentPreprocess.Add([3]string{url, "attachments", msg.ID}, len(url))
	}

	if msg.ChannelId == *ircRevoltChannel && msg.AuthorId != *revoltBotID {
		var nick string = author.Username

		if msg.Masquerade != nil {
			nick = msg.Masquerade.Name
		}

		if !strings.Contains(msg.Content, "\n") {
			if msg.Content != "" {
				mr.ircmsgs <- fmt.Sprintf("%s\\ %s", nick, msg.Content)
			}
		} else {
			for _, line := range strings.Split(msg.Content, "\n") {
				mr.ircmsgs <- fmt.Sprintf("%s\\ %s", nick, line)
			}
		}

		for _, att := range msg.Attachments {
			mr.ircmsgs <- fmt.Sprintf("%s\\ %s", nick, mr.cli.ResolveAttachment(att))
		}
	}

	return nil
}
