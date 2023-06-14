package revolt

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sacOO7/gowebsocket"
)

// New creates a new client with the default Revolt server details.
//
// Use NewWithEndpoint to create a client with a custom endpoint.
func New(token string) *Client {
	return &Client{
		HTTP:    &http.Client{},
		Token:   token,
		BaseURL: "https://api.revolt.chat",
		WSURL:   "wss://ws.revolt.chat",
	}
}

// NewWithEndpoint creates a new client with a custom Revolt endpoint.
//
// You can use this to test the library against an arbirtary Revolt server.
func NewWithEndpoint(token, baseURL, wsURL string) *Client {
	return &Client{
		HTTP:    &http.Client{},
		Token:   token,
		BaseURL: baseURL,
		WSURL:   wsURL,
	}
}

// Client struct.
type Client struct {
	SelfBot *SelfBot
	Token   string
	Socket  gowebsocket.Socket
	HTTP    *http.Client
	Cache   *Cache
	BaseURL string
	WSURL   string
}

// Self bot struct.
type SelfBot struct {
	Email        string `json:"-"`
	Password     string `json:"-"`
	Id           string `json:"id"`
	UserId       string `json:"user_id"`
	SessionToken string `json:"token"`
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
