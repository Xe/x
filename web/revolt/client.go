package revolt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sacOO7/gowebsocket"
)

const (
	WS_URL  = "wss://ws.revolt.chat"
	API_URL = "https://api.revolt.chat"
)

// Client struct.
type Client struct {
	SelfBot *SelfBot
	Token   string
	Socket  gowebsocket.Socket
	HTTP    *http.Client
	Cache   *Cache

	// Event Functions
	OnUnknownEventFunctions       []func(ctx context.Context, message string)
	OnReadyFunctions              []func(ctx context.Context)
	OnMessageFunctions            []func(ctx context.Context, message *Message)
	OnMessageAppendFunctions      []func(ctx context.Context, channelID, messageID string, payload map[string]any)
	OnMessageUpdateFunctions      []func(ctx context.Context, channelID, messageID string, payload map[string]interface{})
	OnMessageDeleteFunctions      []func(ctx context.Context, channelID, messageID string)
	OnChannelCreateFunctions      []func(ctx context.Context, channel *Channel)
	OnChannelUpdateFunctions      []func(ctx context.Context, channelID, clear string, payload map[string]interface{})
	OnChannelDeleteFunctions      []func(ctx context.Context, channelID string)
	OnGroupCreateFunctions        []func(ctx context.Context, group *Group)
	OnGroupMemberAddedFunctions   []func(ctx context.Context, groupID, userID string)
	OnGroupMemberRemovedFunctions []func(ctx context.Context, groupID, userID string)
	OnChannelStartTypingFunctions []func(ctx context.Context, channelID, userID string)
	OnChannelStopTypingFunctions  []func(ctx context.Context, channelID, userID string)
	OnServerCreateFunctions       []func(ctx context.Context, serverID *Server)
	OnServerUpdateFunctions       []func(ctx context.Context, serverID, clear string, payload map[string]interface{})
	OnServerDeleteFunctions       []func(ctx context.Context, serverID string)
	OnServerMemberUpdateFunctions []func(ctx context.Context, serverID, clear string, payload map[string]interface{})
	OnServerMemberJoinFunctions   []func(ctx context.Context, serverID, userID string)
	OnServerMemberLeaveFunctions  []func(ctx context.Context, serverID, userID string)

	// ping timer
	pingMutex sync.Mutex
	lastPing time.Time
}

// Self bot struct.
type SelfBot struct {
	Email        string `json:"-"`
	Password     string `json:"-"`
	Id           string `json:"id"`
	UserId       string `json:"user_id"`
	SessionToken string `json:"token"`
}

// On ready event will run when websocket connection is started and bot is ready to work.
func (c *Client) OnReady(fn func(context.Context)) {
	c.OnReadyFunctions = append(c.OnReadyFunctions, fn)
}

// On message event will run when someone sends a message.
func (c *Client) OnMessage(fn func(ctx context.Context, message *Message)) {
	c.OnMessageFunctions = append(c.OnMessageFunctions, fn)
}

func (c *Client) OnMessageAppend(fn func(ctx context.Context, channelID, messageID string, payload map[string]any)) {
	c.OnMessageAppendFunctions = append(c.OnMessageAppendFunctions, fn)
}

// On message update event will run when someone updates a message.
func (c *Client) OnMessageUpdate(fn func(ctx context.Context, channel_id, message_id string, payload map[string]interface{})) {
	c.OnMessageUpdateFunctions = append(c.OnMessageUpdateFunctions, fn)
}

// On message delete event will run when someone deletes a message.
func (c *Client) OnMessageDelete(fn func(ctx context.Context, channel_id, message_id string)) {
	c.OnMessageDeleteFunctions = append(c.OnMessageDeleteFunctions, fn)
}

// On channel create event will run when someone creates a channel.
func (c *Client) OnChannelCreate(fn func(ctx context.Context, channel *Channel)) {
	c.OnChannelCreateFunctions = append(c.OnChannelCreateFunctions, fn)
}

// On channel update event will run when someone updates a channel.
func (c *Client) OnChannelUpdate(fn func(ctx context.Context, channel_id, clear string, payload map[string]interface{})) {
	c.OnChannelUpdateFunctions = append(c.OnChannelUpdateFunctions, fn)
}

// On channel delete event will run when someone deletes a channel.
func (c *Client) OnChannelDelete(fn func(ctx context.Context, channel_id string)) {
	c.OnChannelDeleteFunctions = append(c.OnChannelDeleteFunctions, fn)
}

// On group channel create event will run when someones creates a group channel.
func (c *Client) OnGroupCreate(fn func(ctx context.Context, group *Group)) {
	c.OnGroupCreateFunctions = append(c.OnGroupCreateFunctions, fn)
}

// On group member added will run when someone is added to a group channel.
func (c *Client) OnGroupMemberAdded(fn func(ctx context.Context, group_id string, user_id string)) {
	c.OnGroupMemberAddedFunctions = append(c.OnGroupMemberAddedFunctions, fn)
}

// On group member removed will run when someone is removed from a group channel.
func (c *Client) OnGroupMemberRemoved(fn func(ctx context.Context, group_id string, user_id string)) {
	c.OnGroupMemberRemovedFunctions = append(c.OnGroupMemberRemovedFunctions, fn)
}

// On unknown event will run when client gets a unknown event.
func (c *Client) OnUnknownEvent(fn func(ctx context.Context, message string)) {
	c.OnUnknownEventFunctions = append(c.OnUnknownEventFunctions, fn)
}

// On channel start typing will run when someone starts to type a message.
func (c *Client) OnChannelStartTyping(fn func(ctx context.Context, channel_id, user_id string)) {
	c.OnChannelStartTypingFunctions = append(c.OnChannelStartTypingFunctions, fn)
}

// On channel stop typing will run when someone stops the typing status.
func (c *Client) OnChannelStopTyping(fn func(ctx context.Context, channel_id, user_id string)) {
	c.OnChannelStopTypingFunctions = append(c.OnChannelStopTypingFunctions, fn)
}

// On server create event will run when someone creates a server.
func (c *Client) OnServerCreate(fn func(ctx context.Context, server *Server)) {
	c.OnServerCreateFunctions = append(c.OnServerCreateFunctions, fn)
}

// On server update will run when someone updates a server.
func (c *Client) OnServerUpdate(fn func(ctx context.Context, server_id, clear string, payload map[string]interface{})) {
	c.OnServerUpdateFunctions = append(c.OnServerUpdateFunctions, fn)
}

// On server delete will run when someone deletes a server.
func (c *Client) OnServerDelete(fn func(ctx context.Context, server_id string)) {
	c.OnServerDeleteFunctions = append(c.OnServerDeleteFunctions, fn)
}

// On server member update will run when a server member updates.
func (c *Client) OnServerMemberUpdate(fn func(ctx context.Context, server_id, clear string, payload map[string]interface{})) {
	c.OnServerMemberUpdateFunctions = append(c.OnServerMemberUpdateFunctions, fn)
}

// On server member join will run when someone joins to the server.
func (c *Client) OnServerMemberJoin(fn func(ctx context.Context, server_id string, user_id string)) {
	c.OnServerMemberJoinFunctions = append(c.OnServerMemberJoinFunctions, fn)
}

// On server member leave will run when someone left from server.
func (c *Client) OnServerMemberLeave(fn func(ctx context.Context, server_id string, user_id string)) {
	c.OnServerMemberLeaveFunctions = append(c.OnServerMemberLeaveFunctions, fn)
}

// Fetch a channel by Id.
func (c *Client) FetchChannel(id string) (*Channel, error) {
	channel := &Channel{}
	channel.Client = c

	data, err := c.Request("GET", "/channels/"+id, []byte{})

	if err != nil {
		return channel, err
	}

	err = json.Unmarshal(data, channel)
	return channel, err
}

// Fetch an user by Id.
func (c *Client) FetchUser(id string) (*User, error) {
	user := &User{}
	user.Client = c

	data, err := c.Request("GET", "/users/"+id, []byte{})

	if err != nil {
		return user, err
	}

	err = json.Unmarshal(data, user)
	return user, err
}

// Fetch a server by Id.
func (c *Client) FetchServer(id string) (*Server, error) {
	server := &Server{}
	server.Client = c

	data, err := c.Request("GET", "/servers/"+id, []byte{})

	if err != nil {
		return server, err
	}

	err = json.Unmarshal(data, server)
	return server, err
}

// Create a server.
func (c *Client) CreateServer(name, description string) (*Server, error) {
	server := &Server{}
	server.Client = c

	data, err := c.Request("POST", "/servers/create", []byte("{\"name\":\""+name+"\",\"description\":\""+description+"\",\"nonce\":\""+genULID()+"\"}"))

	if err != nil {
		return server, err
	}

	err = json.Unmarshal(data, server)
	return server, err
}

// Auth client user.
func (c *Client) Auth(friendlyName string) error {
	if c.SelfBot == nil {
		return fmt.Errorf("can't auth user (not a self-bot.)")
	}

	resp, err := c.Request("POST", "/auth/session/login", []byte("{\"email\":\""+c.SelfBot.Email+"\",\"password\":\""+c.SelfBot.Password+"\",\"friendly_name\":\""+friendlyName+"\"}"))

	if err != nil {
		return err
	}

	err = json.Unmarshal(resp, c.SelfBot)
	return err
}

// Fetch all of the DMs.
func (c *Client) FetchDirectMessages() ([]*Channel, error) {
	var dmChannels []*Channel

	resp, err := c.Request("GET", "/users/dms", []byte{})

	if err != nil {
		return dmChannels, err
	}

	err = json.Unmarshal(resp, &dmChannels)

	if err != nil {
		return dmChannels, err
	}

	// Prepare channels.
	for _, i := range dmChannels {
		i.Client = c
	}

	return dmChannels, nil
}

// Edit client user.
func (c Client) Edit(eu *EditUser) error {
	data, err := json.Marshal(eu)

	if err != nil {
		return err
	}

	_, err = c.Request("PATCH", "/users/@me", data)
	return err
}

// Create a new group.
// Users parameter is a list of users will be added.
func (c *Client) CreateGroup(name, description string, users []string) (*Channel, error) {
	groupChannel := &Channel{}
	groupChannel.Client = c

	dataStruct := &struct {
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Users       []string `json:"users"`
		Nonce       string   `json:"nonce"`
	}{
		Nonce:       genULID(),
		Name:        name,
		Description: description,
		Users:       users,
	}

	data, err := json.Marshal(dataStruct)

	if err != nil {
		return groupChannel, err
	}

	resp, err := c.Request("POST", "/channels/create", data)

	if err != nil {
		return groupChannel, err
	}

	err = json.Unmarshal(resp, groupChannel)
	return groupChannel, err
}

// Fetch relationships.
func (c Client) FetchRelationships() ([]*UserRelations, error) {
	relationshipDatas := []*UserRelations{}

	resp, err := c.Request("GET", "/users/relationships", []byte{})

	if err != nil {
		return relationshipDatas, err
	}

	err = json.Unmarshal(resp, &relationshipDatas)
	return relationshipDatas, err
}

// Send friend request. / Accept friend request.
// User relations struct only will have status. id is not defined for this function.
func (c Client) AddFriend(username string) (*UserRelations, error) {
	relationshipData := &UserRelations{}

	resp, err := c.Request("PUT", "/users/"+username+"/friend", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}

// Deny friend request. / Remove friend.
// User relations struct only will have status. id is not defined for this function.
func (c Client) RemoveFriend(username string) (*UserRelations, error) {
	relationshipData := &UserRelations{}

	resp, err := c.Request("DELETE", "/users/"+username+"/friend", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}

// Create a new bot.
func (c *Client) CreateBot(name string) (*Bot, error) {
	botData := &Bot{}
	botData.Client = c

	resp, err := c.Request("POST", "/bots/create", []byte("{\"name\":\""+name+"\"}"))

	if err != nil {
		return botData, err
	}

	err = json.Unmarshal(resp, botData)
	return botData, err

}

// Fetch client bots.
func (c *Client) FetchBots() (*FetchedBots, error) {
	bots := &FetchedBots{}

	resp, err := c.Request("GET", "/bots/@me", []byte{})

	if err != nil {
		return bots, err
	}

	err = json.Unmarshal(resp, bots)

	if err != nil {
		return bots, err
	}

	// Add client for bots.
	for _, i := range bots.Bots {
		i.Client = c
	}

	// Add client for users.
	for _, i := range bots.Users {
		i.Client = c
	}

	return bots, nil
}

// Fetch a bot.
func (c *Client) FetchBot(id string) (*Bot, error) {
	bot := &struct {
		Bot *Bot `json:"bot"`
	}{
		Bot: &Bot{
			Client: c,
		},
	}

	resp, err := c.Request("GET", "/bots/"+id, []byte{})

	if err != nil {
		return bot.Bot, err
	}

	err = json.Unmarshal(resp, bot)
	return bot.Bot, err
}
