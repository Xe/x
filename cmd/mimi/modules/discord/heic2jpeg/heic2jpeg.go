package heic2jpeg

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"within.website/x/cmd/mimi/internal"
)

func p[T any](v T) *T {
	return &v
}

type Module struct{}

func (m *Module) Register(s *discordgo.Session) {
	s.AddHandler(m.heic2jpeg)
}

func New() *Module {
	return &Module{}
}

func (m *Module) heic2jpeg(s *discordgo.Session, mc *discordgo.MessageCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if len(mc.Attachments) == 0 {
		return
	}

	atts := []*discordgo.MessageAttachment{}

	for _, att := range mc.Attachments {
		switch att.ContentType {
		case "image/heic", "image/avif":
			atts = append(atts, att)
		}
	}

	if len(atts) == 0 {
		return
	}

	os.MkdirAll(filepath.Join(internal.DataDir(), "heic2jpeg"), 0755)
	dir, err := os.MkdirTemp(filepath.Join(internal.DataDir(), "heic2jpeg"), "heic2jpeg")
	if err != nil {
		s.ChannelMessageSend(mc.ChannelID, "failed to create temp dir")
		slog.Error("failed to create temp dir", "err", err)
		return
	}
	defer os.RemoveAll(dir)

	files := make([]*discordgo.File, 0, len(mc.Attachments))

	for _, att := range mc.Attachments {
		// download the image
		req, err := http.NewRequestWithContext(ctx, "GET", att.URL, nil)
		if err != nil {
			s.ChannelMessageSend(mc.ChannelID, "failed to download image")
			slog.Error("failed to download image", "err", err)
			return
		}

		slog.Info("converting", "url", req.URL.String())

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			s.ChannelMessageSend(mc.ChannelID, "failed to download image")
			slog.Error("failed to download image", "err", err)
			return
		}
		defer resp.Body.Close()

		fname := filepath.Join(dir, filepath.Base(req.URL.Path))
		fnameStem := strings.TrimSuffix(fname, filepath.Ext(fname))
		fnameJPEG := fnameStem + ".jpeg"

		fout, err := os.Create(fname)
		if err != nil {
			s.ChannelMessageSend(mc.ChannelID, "failed to save image")
			slog.Error("failed to save image", "err", err)
			return
		}
		defer fout.Close()

		if _, err := io.Copy(fout, resp.Body); err != nil {
			s.ChannelMessageSend(mc.ChannelID, "failed to save image")
			slog.Error("failed to save image", "err", err)
			return
		}

		// convert the image
		cmd := exec.CommandContext(ctx, "magick", fname, "-quality", "80%", fnameJPEG)
		if err := cmd.Run(); err != nil {
			s.ChannelMessageSend(mc.ChannelID, "failed to convert image")
			slog.Error("failed to convert image", "err", err)
			return
		}

		fin, err := os.Open(fnameJPEG)
		if err != nil {
			s.ChannelMessageSend(mc.ChannelID, "failed to open converted image")
			slog.Error("failed to open converted image", "err", err)
			return
		}
		defer fin.Close()

		// queue the image for sending
		files = append(files, &discordgo.File{
			Name:        filepath.Base(fnameJPEG),
			Reader:      fin,
			ContentType: "image/jpeg",
		})
	}

	if _, err := s.ChannelMessageSendComplex(mc.ChannelID, &discordgo.MessageSend{
		Files: files,
		Reference: &discordgo.MessageReference{
			MessageID:       mc.ID,
			ChannelID:       mc.ChannelID,
			GuildID:         mc.GuildID,
			FailIfNotExists: p(true),
		},
	}, discordgo.WithContext(ctx)); err != nil {
		s.ChannelMessageSend(mc.ChannelID, "failed to send converted images")
		slog.Error("failed to send converted images", "err", err)
		return
	}
}
