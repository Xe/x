package revolt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"

	"github.com/sacOO7/gowebsocket"
)

type RevoltSettings struct {
	Revolt   string         `json:"revolt"`
	Features RevoltFeatures `json:"features"`
	Ws       string         `json:"ws"`
	App      string         `json:"app"`
	Vapid    string         `json:"vapid"`
	Build    RevoltBuild    `json:"build"`
}

type RevoltCaptcha struct {
	Enabled bool   `json:"enabled"`
	Key     string `json:"key"`
}

type RevoltAutumn struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

type RevoltJanuary struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

type RevoltVoso struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Ws      string `json:"ws"`
}

type RevoltFeatures struct {
	Captcha    RevoltCaptcha `json:"captcha"`
	Email      bool          `json:"email"`
	InviteOnly bool          `json:"invite_only"`
	Autumn     RevoltAutumn  `json:"autumn"`
	January    RevoltJanuary `json:"january"`
	Voso       RevoltVoso    `json:"voso"`
}

type RevoltBuild struct {
	CommitSha       string `json:"commit_sha"`
	CommitTimestamp string `json:"commit_timestamp"`
	Semver          string `json:"semver"`
	OriginURL       string `json:"origin_url"`
	Timestamp       string `json:"timestamp"`
}

// New creates a new client with the default Revolt server details.
//
// Use NewWithEndpoint to create a client with a custom endpoint.
func New(token string) (*Client, error) {
	var settings RevoltSettings

	resp, err := http.Get("https://api.revolt.chat/")
	if err != nil {
		return nil, fmt.Errorf("failed to get Revolt settings: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode Revolt settings: %w", err)
	}

	return &Client{
		HTTP:     &http.Client{},
		Token:    token,
		BaseURL:  "https://api.revolt.chat",
		WSURL:    "wss://ws.revolt.chat",
		Ticker:   time.NewTicker(3 * time.Second),
		Settings: settings,
	}, nil
}

// NewWithEndpoint creates a new client with a custom Revolt endpoint.
//
// You can use this to test the library against an arbirtary Revolt server.
func NewWithEndpoint(token, baseURL, wsURL string) (*Client, error) {
	var settings RevoltSettings

	resp, err := http.Get(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get Revolt settings: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode Revolt settings: %w", err)
	}

	return &Client{
		HTTP:     &http.Client{},
		Token:    token,
		BaseURL:  baseURL,
		WSURL:    wsURL,
		Ticker:   time.NewTicker(3 * time.Second),
		Settings: settings,
	}, nil
}

// Client struct.
type Client struct {
	SelfBot  *SelfBot
	Token    string
	Socket   gowebsocket.Socket
	HTTP     *http.Client
	Cache    *Cache
	BaseURL  string
	WSURL    string
	Ticker   *time.Ticker
	Settings RevoltSettings
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
func (c *Client) FetchChannel(ctx context.Context, id string) (*Channel, error) {
	channel := &Channel{}

	data, err := c.Request(ctx, "GET", "/channels/"+id, []byte{})

	if err != nil {
		return channel, err
	}

	err = json.Unmarshal(data, channel)
	return channel, err
}

// Fetch an user by Id.
func (c *Client) FetchUser(ctx context.Context, id string) (*User, error) {
	user := &User{}

	data, err := c.Request(ctx, "GET", "/users/"+id, []byte{})

	if err != nil {
		return user, err
	}

	err = json.Unmarshal(data, user)
	return user, err
}

// Fetch a server by Id.
func (c *Client) FetchServer(ctx context.Context, id string) (*Server, error) {
	server := &Server{}

	data, err := c.Request(ctx, "GET", "/servers/"+id, []byte{})

	if err != nil {
		return server, err
	}

	err = json.Unmarshal(data, server)
	return server, err
}

// Create a server.
func (c *Client) CreateServer(ctx context.Context, name, description string) (*Server, error) {
	server := &Server{}

	data, err := c.Request(ctx, "POST", "/servers/create", []byte("{\"name\":\""+name+"\",\"description\":\""+description+"\",\"nonce\":\""+genULID()+"\"}"))

	if err != nil {
		return server, err
	}

	err = json.Unmarshal(data, server)
	return server, err
}

// Auth client user.
func (c *Client) Auth(ctx context.Context, friendlyName string) error {
	if c.SelfBot == nil {
		return fmt.Errorf("can't auth user (not a self-bot.)")
	}

	resp, err := c.Request(ctx, "POST", "/auth/session/login", []byte("{\"email\":\""+c.SelfBot.Email+"\",\"password\":\""+c.SelfBot.Password+"\",\"friendly_name\":\""+friendlyName+"\"}"))

	if err != nil {
		return err
	}

	err = json.Unmarshal(resp, c.SelfBot)
	return err
}

// Fetch all of the DMs.
func (c *Client) FetchDirectMessages(ctx context.Context) ([]*Channel, error) {
	var dmChannels []*Channel

	resp, err := c.Request(ctx, "GET", "/users/dms", []byte{})

	if err != nil {
		return dmChannels, err
	}

	err = json.Unmarshal(resp, &dmChannels)

	if err != nil {
		return dmChannels, err
	}

	return dmChannels, nil
}

// Edit client user.
func (c Client) Edit(ctx context.Context, eu *EditUser) error {
	data, err := json.Marshal(eu)

	if err != nil {
		return err
	}

	_, err = c.Request(ctx, "PATCH", "/users/@me", data)
	return err
}

// Create a new group.
// Users parameter is a list of users will be added.
func (c *Client) CreateGroup(ctx context.Context, name, description string, users []string) (*Channel, error) {
	groupChannel := &Channel{}

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

	resp, err := c.Request(ctx, "POST", "/channels/create", data)

	if err != nil {
		return groupChannel, err
	}

	err = json.Unmarshal(resp, groupChannel)
	return groupChannel, err
}

// Fetch relationships.
func (c Client) FetchRelationships(ctx context.Context) ([]*UserRelations, error) {
	relationshipDatas := []*UserRelations{}

	resp, err := c.Request(ctx, "GET", "/users/relationships", []byte{})

	if err != nil {
		return relationshipDatas, err
	}

	err = json.Unmarshal(resp, &relationshipDatas)
	return relationshipDatas, err
}

// Send friend request. / Accept friend request.
// User relations struct only will have status. id is not defined for this function.
func (c Client) AddFriend(ctx context.Context, username string) (*UserRelations, error) {
	relationshipData := &UserRelations{}

	resp, err := c.Request(ctx, "PUT", "/users/"+username+"/friend", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}

// Deny friend request. / Remove friend.
// User relations struct only will have status. id is not defined for this function.
func (c Client) RemoveFriend(ctx context.Context, username string) (*UserRelations, error) {
	relationshipData := &UserRelations{}

	resp, err := c.Request(ctx, "DELETE", "/users/"+username+"/friend", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}

// Create a new bot.
func (c *Client) CreateBot(ctx context.Context, name string) (*Bot, error) {
	botData := &Bot{}

	resp, err := c.Request(ctx, "POST", "/bots/create", []byte("{\"name\":\""+name+"\"}"))

	if err != nil {
		return botData, err
	}

	err = json.Unmarshal(resp, botData)
	return botData, err

}

// Fetch client bots.
func (c *Client) FetchBots(ctx context.Context) (*FetchedBots, error) {
	bots := &FetchedBots{}

	resp, err := c.Request(ctx, "GET", "/bots/@me", []byte{})

	if err != nil {
		return bots, err
	}

	err = json.Unmarshal(resp, bots)

	if err != nil {
		return bots, err
	}

	return bots, nil
}

// Fetch a bot.
func (c *Client) FetchBot(ctx context.Context, id string) (*Bot, error) {
	bot := &struct {
		Bot *Bot `json:"bot"`
	}{
		Bot: &Bot{},
	}

	resp, err := c.Request(ctx, "GET", "/bots/"+id, []byte{})

	if err != nil {
		return bot.Bot, err
	}

	err = json.Unmarshal(resp, bot)
	return bot.Bot, err
}

// Upload a file to Revolt using a multi-part form.
func (c *Client) Upload(ctx context.Context, tag, fname string, data []byte) (string, error) {
	type response struct {
		ID string `json:"id"`
	}

	// Create a new multi-part form.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fname))
	header.Set("Content-Type", http.DetectContentType(data))

	// Add the file to the form.
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	// Close the form and set the content type.
	err = writer.Close()
	if err != nil {
		return "", err
	}

	// Send the request to the server.
	resp, err := c.RequestWithPathAndContentType(ctx, "POST", c.Settings.Features.Autumn.URL+"/"+tag, writer.FormDataContentType(), body.Bytes())
	if err != nil {
		return "", err
	}

	// Parse the response and return the ID of the uploaded file.
	var res response
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return "", err
	}
	return res.ID, nil
}

func (c *Client) ResolveAttachment(att *Attachment) string {
	return c.Settings.Features.Autumn.URL + "/" + att.Tag + "/" + att.ID
}
