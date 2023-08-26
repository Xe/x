package revolt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"nhooyr.io/websocket"
)

func (c *Client) Connect(ctx context.Context, handler Handler) {
	lg := slog.Default().With("at", "websocket-client")

	go func(ctx context.Context) {
		if err := c.doWebsocket(ctx, c.Token, c.WSURL, handler); err != nil {
			lg.Error("websocket error, retrying", "err", err)
		}

		t := time.NewTicker(30 * time.Second)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				lg.Debug("reconnecting")
				if err := c.doWebsocket(ctx, c.Token, c.WSURL, handler); err != nil {
					lg.Error("websocket error, retrying", "err", err)
				}
			}
		}
	}(ctx)
}

func (c *Client) doWebsocket(ctx context.Context, token, wsURL string, handler Handler) error {
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{})
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "doWebsocket function returned")
	slog.Debug("connected to websocket", "server", wsURL)

	data, err := json.Marshal(struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}{
		Type:  "Authenticate",
		Token: token,
	})
	if err != nil {
		return err
	}

	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				data, err := json.Marshal(struct {
					Type string `json:"type"`
					Data int    `json:"data"`
				}{
					Type: "Ping",
					Data: 0,
				})
				if err != nil {
					slog.Error("can't marshal ping message", "err", err)
					continue
				}
				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					slog.Error("can't write ping message", "err", err)
					continue
				}
			}
		}
	}(ctx)

	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		return err
	}

	lastMsgSeen := time.Now()
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if time.Since(lastMsgSeen) > 5*time.Minute {
					conn.Close(websocket.StatusNormalClosure, "ping timeout")
					return
				}
			}
		}
	}(ctx)

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if typ != websocket.MessageText {
			return fmt.Errorf("unexpected message type: %v", typ)
		}
		lastMsgSeen = time.Now()

		if err := c.handleOneMessage(ctx, data, handler); err != nil {
			return err
		}
	}
}

func (c *Client) handleOneMessage(ctx context.Context, data []byte, handler Handler) error {
	var msg typeResolver
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	switch msg.Type {
	case "Pong":
	case "Authenticated":
		if err := handler.Authenticated(ctx); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "Ready":
		if err := handler.Ready(ctx); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "Error":
		var wserr WSError
		if err := json.Unmarshal(data, &wserr); err != nil {
			return err
		}
		return wserr
	case "Bulk":
		var bulk struct {
			Type     string            `json:"type"`
			Messages []json.RawMessage `json:"v"`
		}
		if err := json.Unmarshal(data, &bulk); err != nil {
			return err
		}
		for _, msg := range bulk.Messages {
			if err := c.handleOneMessage(ctx, msg, handler); err != nil {
				return err
			}
		}
		return nil
	case "Message":
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := handler.MessageCreate(ctx, &msg); err != nil {
			slog.Error("error in handler", "call", "Message", "err", err)
		}
	case "MessageUpdate":
		var msg struct {
			Type      string   `json:"type"`
			MessageID string   `json:"id"`
			ChannelID string   `json:"channel"`
			Data      *Message `json:"data"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := handler.MessageUpdate(ctx, msg.ChannelID, msg.MessageID, msg.Data); err != nil {
			slog.Error("error in handler.MessageUpdate", "err", err)
		}
	case "MessageAppend":
		var msg struct {
			Type      string         `json:"type"`
			MessageID string         `json:"id"`
			ChannelID string         `json:"channel"`
			Append    *MessageAppend `json:"append"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := handler.MessageAppend(ctx, msg.ChannelID, msg.MessageID, msg.Append); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "MessageDelete":
		var msg struct {
			Type      string `json:"type"`
			MessageID string `json:"id"`
			ChannelID string `json:"channel"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := handler.MessageDelete(ctx, msg.ChannelID, msg.MessageID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "MessageReact":
		var msg struct {
			Type      string `json:"type"`
			MessageID string `json:"id"`
			ChannelID string `json:"channel_id"`
			UserID    string `json:"user_id"`
			Emoji     string `json:"emoji_id"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := handler.MessageReact(ctx, msg.ChannelID, msg.MessageID, msg.UserID, msg.Emoji); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "MessageUnreact":
		var msg struct {
			Type      string `json:"type"`
			MessageID string `json:"id"`
			ChannelID string `json:"channel_id"`
			UserID    string `json:"user_id"`
			Emoji     string `json:"emoji_id"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := handler.MessageUnreact(ctx, msg.ChannelID, msg.MessageID, msg.UserID, msg.Emoji); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "MessageRemoveReaction":
		var msg struct {
			Type      string `json:"type"`
			MessageID string `json:"id"`
			ChannelID string `json:"channel_id"`
			EmojiID   string `json:"emoji_id"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := handler.MessageRemoveReaction(ctx, msg.ChannelID, msg.MessageID, msg.EmojiID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ChannelCreate":
		var ch Channel
		if err := json.Unmarshal(data, &ch); err != nil {
			return err
		}
		if err := handler.ChannelCreate(ctx, &ch); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ChannelUpdate":
		var ch struct {
			Type      string   `json:"type"`
			ChannelID string   `json:"id"`
			Data      Channel  `json:"data"`
			Clear     []string `json:"clear"`
		}
		if err := json.Unmarshal(data, &ch); err != nil {
			return err
		}
		if err := handler.ChannelUpdate(ctx, ch.ChannelID, &ch.Data, ch.Clear); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ChannelDelete":
		var ch struct {
			Type      string `json:"type"`
			ChannelID string `json:"id"`
		}
		if err := json.Unmarshal(data, &ch); err != nil {
			return err
		}
		if err := handler.ChannelDelete(ctx, ch.ChannelID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ChannelAck":
		var ch struct {
			Type      string `json:"type"`
			ChannelID string `json:"id"`
			UserID    string `json:"user_id"`
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(data, &ch); err != nil {
			return err
		}
		if err := handler.ChannelAck(ctx, ch.ChannelID, ch.UserID, ch.MessageID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ChannelStartTyping":
		var ch struct {
			Type      string `json:"type"`
			ChannelID string `json:"id"`
			UserID    string `json:"user_id"`
		}
		if err := json.Unmarshal(data, &ch); err != nil {
			return err
		}
		if err := handler.ChannelStartTyping(ctx, ch.ChannelID, ch.UserID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ChannelStopTyping":
		var ch struct {
			Type      string `json:"type"`
			ChannelID string `json:"id"`
			UserID    string `json:"user_id"`
		}
		if err := json.Unmarshal(data, &ch); err != nil {
			return err
		}
		if err := handler.ChannelStopTyping(ctx, ch.ChannelID, ch.UserID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ChannelGroupJoin":
		var ch struct {
			Type      string `json:"type"`
			ChannelID string `json:"id"`
			UserID    string `json:"user_id"`
		}
		if err := json.Unmarshal(data, &ch); err != nil {
			return err
		}
		if err := handler.ChannelGroupJoin(ctx, ch.ChannelID, ch.UserID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ChannelGroupLeave":
		var ch struct {
			Type      string `json:"type"`
			ChannelID string `json:"id"`
			UserID    string `json:"user_id"`
		}
		if err := json.Unmarshal(data, &ch); err != nil {
			return err
		}
		if err := handler.ChannelGroupLeave(ctx, ch.ChannelID, ch.UserID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ServerCreate":
		var srv Server
		if err := json.Unmarshal(data, &srv); err != nil {
			return err
		}
		if err := handler.ServerCreate(ctx, &srv); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ServerUpdate":
		var srv struct {
			Type     string   `json:"type"`
			ServerID string   `json:"id"`
			Data     Server   `json:"data"`
			Clear    []string `json:"clear"`
		}
		if err := json.Unmarshal(data, &srv); err != nil {
			return err
		}
		if err := handler.ServerUpdate(ctx, srv.ServerID, &srv.Data, srv.Clear); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ServerDelete":
		var srv struct {
			Type     string `json:"type"`
			ServerID string `json:"id"`
		}
		if err := json.Unmarshal(data, &srv); err != nil {
			return err
		}
		if err := handler.ServerDelete(ctx, srv.ServerID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ServerMemberUpdate":
		var srv struct {
			Type string `json:"type"`
			ID   struct {
				Server string `json:"server"`
				User   string `json:"user"`
			} `json:"id"`
			Data  Member   `json:"data"`
			Clear []string `json:"clear"`
		}
		if err := json.Unmarshal(data, &srv); err != nil {
			return err
		}
		if err := handler.ServerMemberUpdate(ctx, srv.ID.Server, srv.ID.User, &srv.Data, srv.Clear); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ServerMemberJoin":
		var srv struct {
			Type     string `json:"type"`
			ServerID string `json:"id"`
			UserID   string `json:"user"`
		}
		if err := json.Unmarshal(data, &srv); err != nil {
			return err
		}
		if err := handler.ServerMemberJoin(ctx, srv.ServerID, srv.UserID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ServerMemberLeave":
		var srv struct {
			Type     string `json:"type"`
			ServerID string `json:"id"`
			UserID   string `json:"user"`
		}
		if err := json.Unmarshal(data, &srv); err != nil {
			return err
		}
		if err := handler.ServerMemberLeave(ctx, srv.ServerID, srv.UserID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "ServerRoleUpdate":
		var srv struct {
			Type     string   `json:"type"`
			ServerID string   `json:"id"`
			RoleID   string   `json:"role_id"`
			Data     Role     `json:"data"`
			Clear    []string `json:"clear"`
		}
		if err := json.Unmarshal(data, &srv); err != nil {
			return err
		}
		if err := handler.ServerRoleUpdate(ctx, srv.ServerID, srv.RoleID, &srv.Data, srv.Clear); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "UserUpdate":
		var usr struct {
			Type   string   `json:"type"`
			UserID string   `json:"id"`
			Data   User     `json:"data"`
			Clear  []string `json:"clear"`
		}
		if err := json.Unmarshal(data, &usr); err != nil {
			return err
		}
		if err := handler.UserUpdate(ctx, usr.UserID, &usr.Data, usr.Clear); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "UserRelationship":
		var usr struct {
			Type   string `json:"type"`
			UserID string `json:"id"`
			User   User   `json:"data"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &usr); err != nil {
			return err
		}
		if err := handler.UserRelationship(ctx, usr.UserID, &usr.User, usr.Status); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "UserPlatformWipe":
		var usr struct {
			Type   string `json:"type"`
			UserID string `json:"id"`
			Flags  string `json:"flags"`
		}
		if err := json.Unmarshal(data, &usr); err != nil {
			return err
		}
		if err := handler.UserPlatformWipe(ctx, usr.UserID, usr.Flags); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "EmojiCreate":
		var emoji Emoji
		if err := json.Unmarshal(data, &emoji); err != nil {
			return err
		}
		if err := handler.EmojiCreate(ctx, &emoji); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	case "EmojiDelete":
		var emj struct {
			Type    string `json:"type"`
			EmojiID string `json:"id"`
		}
		if err := json.Unmarshal(data, &emj); err != nil {
			return err
		}
		if err := handler.EmojiDelete(ctx, emj.EmojiID); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	default:
		if err := handler.UnknownEvent(ctx, msg.Type, data); err != nil {
			slog.Error("error in handler", "call", msg.Type, "err", err)
		}
	}
	return nil
}

type typeResolver struct {
	Type string `json:"type"`
}

type WSError struct {
	Type   string `json:"type"`
	ErrMsg string `json:"error"`
}

func (e WSError) Error() string {
	return e.ErrMsg
}
