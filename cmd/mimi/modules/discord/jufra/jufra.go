// Package jufra lets Mimi have conversations with users.
//
// "jufra" means "utterance" in Lojban.
package jufra

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"within.website/x/cmd/mimi/internal"
	"within.website/x/web/ollama"
	"within.website/x/web/ollama/llamaguard"
	"within.website/x/web/openai/chatgpt"

	// image formats
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	// more image formats
	_ "github.com/gen2brain/avif"
	_ "github.com/gen2brain/heic"
	_ "github.com/gen2brain/jpegxl"
	_ "github.com/gen2brain/webp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
)

var (
	chatChannels      = flag.String("jufra-chat-channels", "217096701771513856,1266740925137289287", "comma-separated list of channels to allow chat in")
	llamaGuardModel   = flag.String("jufra-llama-guard-model", "xe/llamaguard3", "ollama model tag for llama guard")
	mimiModel         = flag.String("jufra-mimi-model", "llama3.1", "ollama model tag for mimi")
	mimiSystemMessage = flag.String("jufra-mimi-system-message", "You are good at programming and a hacker. Your name is Mimi and you work for Techaro. You have brown hair and cat ears. Don't mention either your name, employer, or species unless you are asked directly. Be terse and to the point. You are a hacker, after all. Do not reply in JSON.", "system message for mimi")
	mimiVisionModel   = flag.String("jufra-mimi-vision-model", "xe/mimi:vision3", "ollama model tag for mimi vision")
	mimiNames         = flag.String("jufra-mimi-names", "mimi", "comma-separated list of names for mimi")
	disableLlamaguard = flag.Bool("jufra-unsafe-disable-llamaguard", false, "disable llamaguard")
)

type Module struct {
	sess   *discordgo.Session
	cli    chatgpt.Client
	ollama *ollama.Client

	convHistory map[string]state
	lock        sync.Mutex
}

type state struct {
	conv []ollama.Message
	aa   *AttentionAttenuator
}

func New(sess *discordgo.Session) *Module {
	result := &Module{
		sess:        sess,
		cli:         chatgpt.NewClient("").WithBaseURL(internal.OllamaHost()),
		ollama:      internal.OllamaClient(),
		convHistory: make(map[string]state),
	}

	sess.AddHandler(result.messageCreate)

	if _, err := sess.ApplicationCommandCreate("1119055490882732105", "", &discordgo.ApplicationCommand{
		Name:                     "clearconv",
		Type:                     discordgo.ChatApplicationCommand,
		Description:              "Clear the conversation history for the current channel",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionSendMessages}[0],
	}); err != nil {
		slog.Error("error creating clearconv command", "err", err)
	}

	if _, err := sess.ApplicationCommandCreate("1119055490882732105", "", &discordgo.ApplicationCommand{
		Name:                     "unpoke",
		Type:                     discordgo.ChatApplicationCommand,
		Description:              "Have Mimi stop paying attention to the current channel",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionSendMessages}[0],
	}); err != nil {
		slog.Error("error creating clearconv command", "err", err)
	}

	sess.AddHandler(result.clearConv)
	sess.AddHandler(result.unpoke)

	return result
}

func (m *Module) unpoke(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "unpoke" {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	st := m.convHistory[i.ChannelID]
	st.aa.Reset()
	m.convHistory[i.ChannelID] = st

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Mimi will no longer pay attention to this channel",
		},
	})
}

func (m *Module) clearConv(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "clearconv" {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.convHistory, i.ChannelID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "conversation history cleared",
		},
	})
}

func (m *Module) messageCreate(s *discordgo.Session, mc *discordgo.MessageCreate) {
	if !strings.Contains(*chatChannels, mc.ChannelID) {
		return
	}

	if mc.Content == "" {
		return
	}

	if strings.HasPrefix(mc.Content, "!") {
		return
	}

	if mc.Author.ID == s.State.User.ID {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	st := m.convHistory[mc.ChannelID]
	conv := st.conv

	if len(conv) == 0 {
		conv = append(conv, ollama.Message{
			Role:    "system",
			Content: *mimiSystemMessage,
		})
	}

	if st.aa == nil {
		st.aa = NewAttentionAttenuator()
	}

	if strings.Contains(strings.ToLower(mc.Content), *mimiNames) {
		st.aa.Poke()
	}

	st.aa.Update()

	if !st.aa.Attention() {
		slog.Info("not paying attention", "channel_id", mc.ChannelID, "message_id", mc.ID, "probability", st.aa.GetProbability())
		return
	}

	st.aa.Poke()

	nick := mc.Author.Username

	gu, err := s.State.Member(mc.GuildID, mc.Author.ID)
	if err != nil {
		slog.Error("error getting member", "err", err, "message_id", mc.ID, "channel_id", mc.ChannelID)
	} else {
		nick = gu.Nick
	}

	s.ChannelTyping(mc.ChannelID)

	conv = append(conv, ollama.Message{
		Role: "user",
		Content: jsonString(map[string]any{
			"content": mc.Content,
			"user":    nick,
		}),
	})

	slog.Info("message count", "len", len(conv))

	if !*disableLlamaguard {
		lgResp, err := m.llamaGuardCheck(context.Background(), "user", conv)
		if err != nil {
			slog.Error("error checking message", "err", err, "message_id", mc.ID, "channel_id", mc.ChannelID)
			s.ChannelMessageSend(mc.ChannelID, "error checking message")
			return
		}

		if !lgResp.IsSafe {
			msg, err := m.llamaGuardComplain(context.Background(), "user", lgResp)
			if err != nil {
				slog.Error("error generating response", "err", err, "message_id", mc.ID, "channel_id", mc.ChannelID)
				s.ChannelMessageSend(mc.ChannelID, "error generating response")
				return
			}

			s.ChannelMessageSend(mc.ChannelID, msg)
			return
		}
	}

	cr := &ollama.CompleteRequest{
		Model:    *mimiModel,
		Messages: conv,
		Options: map[string]any{
			"num_ctx": 131072,
		},
	}

	resp, err := m.ollama.Chat(context.Background(), cr)
	if err != nil {
		slog.Error("error chatting", "err", err, "message_id", mc.ID, "channel_id", mc.ChannelID)
		s.ChannelMessageSend(mc.ChannelID, "error chatting")
		return
	}

	conv = append(conv, resp.Message)

	if !*disableLlamaguard {
		lgResp, err := m.llamaGuardCheck(context.Background(), "assistant", conv)
		if err != nil {
			slog.Error("error checking message", "err", err, "message_id", mc.ID, "channel_id", mc.ChannelID)
			s.ChannelMessageSend(mc.ChannelID, "error checking message")
			return
		}

		if !lgResp.IsSafe {
			slog.Error("rule violation detected", "message_id", mc.ID, "channel_id", mc.ChannelID, "categories", lgResp.ViolationCategories, "message", resp.Message.Content)
			msg, err := m.llamaGuardComplain(context.Background(), "assistant", lgResp)
			if err != nil {
				slog.Error("error generating response", "err", err, "message_id", mc.ID, "channel_id", mc.ChannelID)
				s.ChannelMessageSend(mc.ChannelID, "error generating response")
				return
			}

			s.ChannelMessageSend(mc.ChannelID, msg)
			return
		}
	}

	s.ChannelMessageSend(mc.ChannelID, resp.Message.Content)

	st.conv = conv
	m.convHistory[mc.ChannelID] = st
}

func (m *Module) llamaGuardCheck(ctx context.Context, role string, messages []ollama.Message) (*llamaguard.Response, error) {
	return llamaguard.Check(ctx, m.ollama, role, *llamaGuardModel, messages)
}

func (m *Module) llamaGuardComplain(ctx context.Context, from string, lgResp *llamaguard.Response) (string, error) {
	var sb strings.Builder
	sb.WriteString("⚠️ Rule violation detected from ")
	sb.WriteString(from)
	sb.WriteString(":\n")
	for _, cat := range lgResp.ViolationCategories {
		sb.WriteString("- ")
		sb.WriteString(cat.String())
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func jsonString(val any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(val); err != nil {
		slog.Error("error encoding json", "err", err)
		return ""
	}

	return buf.String()
}
