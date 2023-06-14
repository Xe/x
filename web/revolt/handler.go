package revolt

import "context"

type NullHandler struct{}

func (NullHandler) UnknownEvent(context.Context, string, []byte) error { return nil }
func (NullHandler) Ready(context.Context) error                        { return nil }
func (NullHandler) MessageCreate(context.Context, *Message) error      { return nil }
func (NullHandler) MessageAppend(context.Context, string, string, *MessageAppend) error {
	return nil
}
func (NullHandler) MessageDelete(context.Context, string, string) error { return nil }
func (NullHandler) MessageUpdate(context.Context, string, string, *Message) error {
	return nil
}
func (NullHandler) MessageReact(context.Context, string, string, string, string) error {
	return nil
}
func (NullHandler) MessageUnreact(context.Context, string, string, string, string) error {
	return nil
}
func (NullHandler) MessageRemoveReaction(context.Context, string, string, string) error {
	return nil
}
func (NullHandler) ChannelCreate(context.Context, *Channel) error { return nil }
func (NullHandler) ChannelUpdate(context.Context, string, string, *Channel) error {
	return nil
}
func (NullHandler) ChannelDelete(context.Context, string) error { return nil }
func (NullHandler) ChannelAck(context.Context, string, string, string) error {
	return nil
}
func (NullHandler) ChannelStartTyping(context.Context, string, string) error {
	return nil
}
func (NullHandler) ChannelStopTyping(context.Context, string, string) error {
	return nil
}
func (NullHandler) ChannelGroupJoin(context.Context, string, string) error {
	return nil
}
func (NullHandler) ChannelGroupLeave(context.Context, string, string) error {
	return nil
}
func (NullHandler) ServerCreate(context.Context, *Server) error { return nil }
func (NullHandler) ServerUpdate(context.Context, string, *Server, []string) error {
	return nil
}
func (NullHandler) ServerDelete(context.Context, string) error { return nil }
func (NullHandler) ServerMemberJoin(context.Context, string, string) error {
	return nil
}
func (NullHandler) ServerMemberLeave(context.Context, string, string) error {
	return nil
}
func (NullHandler) ServerMemberUpdate(context.Context, string, string, *Member, []string) error {
	return nil
}
func (NullHandler) ServerRoleCreate(context.Context, string, *Role) error {
	return nil
}
func (NullHandler) ServerRoleUpdate(context.Context, string, string, *Role, []string) error {
	return nil
}
func (NullHandler) ServerRoleDelete(context.Context, string, string) error {
	return nil
}
func (NullHandler) ServerChannelCreate(context.Context, string, *Channel) error {
	return nil
}
func (NullHandler) ServerChannelUpdate(context.Context, string, string, *Channel, []string) error {
	return nil
}
func (NullHandler) ServerChannelDelete(context.Context, string, string) error {
	return nil
}
func (NullHandler) UserUpdate(context.Context, string, *User, []string) error { return nil }
func (NullHandler) UserRelationship(context.Context, string, *User, string) error {
	return nil
}
func (NullHandler) UserPlatformWipe(context.Context, string, string) error {
	return nil
}
func (NullHandler) EmojiCreate(context.Context, *Emoji) error {
	return nil
}
func (NullHandler) EmojiDelete(context.Context, string) error {
	return nil
}

type Handler interface {
	UnknownEvent(ctx context.Context, kind string, data []byte) error
	Ready(context.Context) error

	// Messages

	MessageCreate(ctx context.Context, message *Message) error
	MessageAppend(ctx context.Context, channelID, messageID string, data *MessageAppend) error
	MessageDelete(ctx context.Context, channelID, messageID string) error
	MessageUpdate(ctx context.Context, channelID, messageID string, data *Message) error
	MessageReact(ctx context.Context, messageID, channelID, userID, emojiID string) error
	MessageUnreact(ctx context.Context, messageID, channelID, userID, emojiID string) error
	MessageRemoveReaction(ctx context.Context, messageID, channelID, emojiID string) error

	// Channels

	ChannelCreate(ctx context.Context, channel *Channel) error
	ChannelUpdate(ctx context.Context, channelID, clear string, channel *Channel) error
	ChannelDelete(ctx context.Context, channelID string) error
	ChannelAck(ctx context.Context, channelID, userID, messageID string) error
	ChannelStartTyping(ctx context.Context, channelID, userID string) error
	ChannelStopTyping(ctx context.Context, channelID, userID string) error

	// User groups

	ChannelGroupJoin(ctx context.Context, channelID, userID string) error
	ChannelGroupLeave(ctx context.Context, channelID, userID string) error

	// Servers

	ServerCreate(ctx context.Context, srv *Server) error
	ServerUpdate(ctx context.Context, serverID string, srv *Server, clear []string) error
	ServerDelete(ctx context.Context, serverID string) error
	ServerMemberUpdate(ctx context.Context, serverID, userID string, data *Member, clear []string) error
	ServerMemberJoin(ctx context.Context, serverID, userID string) error
	ServerMemberLeave(ctx context.Context, serverID, userID string) error
	ServerRoleUpdate(ctx context.Context, serverID, roleID string, role *Role, clear []string) error

	// Users
	UserUpdate(ctx context.Context, userID string, user *User, clear []string) error
	UserRelationship(ctx context.Context, userID string, user *User, status string) error
	UserPlatformWipe(ctx context.Context, userID string, flags string) error

	// Emoji
	EmojiCreate(ctx context.Context, emoji *Emoji) error
	EmojiDelete(ctx context.Context, emojiID string) error
}
