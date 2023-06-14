package revolt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sacOO7/gowebsocket"
	"within.website/ln"
	"within.website/ln/opname"
)

func (c *Client) Start(ctx context.Context) {
	// Create new socket
	c.Socket = gowebsocket.New(WS_URL)
	c.HTTP = &http.Client{}

	// Auth the user if self-bot.
	// if c.SelfBot != nil {
	// 	c.Auth()
	// }

	// Send auth when connected
	c.Socket.OnConnected = func(_ gowebsocket.Socket) {
		c.handleWebsocketAuth()
	}

	c.Socket.OnTextMessage = func(message string, _ gowebsocket.Socket) {
		//fmt.Println(message)

		// Parse data
		rawData := &struct {
			Type string `json:"type"`
		}{}
		err := json.Unmarshal([]byte(message), rawData)

		if err != nil {
			c.Close()
			panic(err)
		}

		if rawData.Type == "Authenticated" {
			go c.ping()
		}

		// Handle events.
		c.handleEvents(ctx, rawData, message)
		// fmt.Println(message)
	}

	c.Socket.OnDisconnected = func (err error, sock gowebsocket.Socket) {
		ctx := opname.With(ctx, "websocket-ondisconnect")
		ln.Error(ctx, err)
		sock.Conn.Close()
	}

	// Start connection.
	c.Socket.Connect()

	go func() {
		<-ctx.Done()
		c.Close()
	}()
}

// Handle on connected.
func (c *Client) handleWebsocketAuth() {
	if c.SelfBot == nil {
		c.Socket.SendText(fmt.Sprintf("{\"type\":\"Authenticate\",\"token\":\"%s\"}", c.Token))
	} else {
		c.Socket.SendText(fmt.Sprintf("{\"type\":\"Authenticate\",\"result\":\"Success\",\"_id\":\"%s\",\"token\":\"%s\",\"user_id\":\"%s\",\"name\":\"revolt\"}", c.SelfBot.Id, c.SelfBot.SessionToken, c.SelfBot.UserId))
	}
}

// Close the websocket and clean up associated resources.
func (c *Client) Close() {
	c.Socket.Close()
}

// Ping websocket.
func (c *Client) ping() {
	for {
		time.Sleep(30 * time.Second)
		c.Socket.SendText("{\"type\":\"Ping\",\"data\":0}")
	}
}

// Handle events.
func (c *Client) handleEvents(ctx context.Context, rawData *struct {
	Type string `json:"type"`
}, message string) {
	type junk struct {
		Channel string `json:"channel"`
		ID      string `json:"id"`
		User    string `json:"user"`
	}

	ctx = opname.With(ctx, "handleEvents."+rawData.Type)
	
	switch rawData.Type {
	case "Pong", "Authenticated": // ignore these messages
	case "Ready":
		// Add cache.
		c.handleCache(message)

		// onReady event
		if c.OnReadyFunctions != nil {
			for _, i := range c.OnReadyFunctions {
				i(ctx)
			}
		}
	case "Message":
		// Message create event.
		msgData := &Message{}
		msgData.Client = c

		if err := json.Unmarshal([]byte(message), msgData); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnMessageFunctions {
			i(ctx, msgData)
		}
	case "MessageAppend":
		data := struct{
			ChannelId string                 `json:"channel"`
			MessageId string                 `json:"id"`
			Append   map[string]interface{} `json:"append"`
		}{}

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			ln.Error(ctx, err, ln.F{"type": rawData.Type})
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnMessageAppendFunctions {
			i(ctx, data.ChannelId, data.MessageId, data.Append)
		}
	case "MessageUpdate":
		// Message update event.
		data := &struct {
			ChannelId string                 `json:"channel"`
			MessageId string                 `json:"id"`
			Payload   map[string]interface{} `json:"data"`
		}{}

		if err := json.Unmarshal([]byte(message), data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnMessageUpdateFunctions {
			i(ctx, data.ChannelId, data.MessageId, data.Payload)
		}
	case "MessageDelete":
		// Message delete event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnMessageDeleteFunctions {
			i(ctx, data.Channel, data.ID)
		}
	case "ChannelCreate":
		// Channel create event.
		channelData := &Channel{}
		channelData.Client = c

		if err := json.Unmarshal([]byte(message), channelData); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelCreateFunctions {
			i(ctx, channelData)
		}
	case "ChannelUpdate":
		// Channel update event.
		data := &struct {
			ChannelId string                 `json:"id"`
			Clear     string                 `json:"clear"`
			Payload   map[string]interface{} `json:"data"`
		}{}

		if err := json.Unmarshal([]byte(message), data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelUpdateFunctions {
			i(ctx, data.ChannelId, data.Clear, data.Payload)
		}
	case "ChannelDelete":
		// Channel delete event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelDeleteFunctions {
			i(ctx, data.ID)
		}
	case "GroupCreate":
		// Group channel create event.
		groupChannelData := &Group{}
		groupChannelData.Client = c

		if err := json.Unmarshal([]byte(message), groupChannelData); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnGroupCreateFunctions {
			i(ctx, groupChannelData)
		}
	case "GroupMemeberAdded":
		// Group member added event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnGroupMemberAddedFunctions {
			i(ctx, data.ID, data.User)
		}
	case "GroupMemberRemoved":
		// Group member removed event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnGroupMemberRemovedFunctions {
			i(ctx, data.ID, data.User)
		}
	case "ChannelStartTyping":
		// Channel start typing event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelStartTypingFunctions {
			i(ctx, data.ID, data.User)
		}
	case "ChannelStopTyping":
		// Channel stop typing event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelStopTypingFunctions {
			i(ctx, data.ID, data.User)
		}
	case "ServerCreate":
		// Server create event.
		serverData := &Server{}
		serverData.Client = c

		if err := json.Unmarshal([]byte(message), serverData); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerCreateFunctions {
			i(ctx, serverData)
		}
	case "ServerUpdate":
		// Server update event.
		data := &struct {
			ServerId string                 `json:"id"`
			Clear    string                 `json:"clear"`
			Payload  map[string]interface{} `json:"data"`
		}{}

		if err := json.Unmarshal([]byte(message), data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerUpdateFunctions {
			i(ctx, data.ServerId, data.Clear, data.Payload)
		}
	case "ServerDelete":
		// Server delete event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerDeleteFunctions {
			i(ctx, data.ID)
		}
	case "ServerMemberUpdate":
		// Member update event.
		data := &struct {
			ServerId string                 `json:"id"`
			Clear    string                 `json:"clear"`
			Payload  map[string]interface{} `json:"data"`
		}{}

		if err := json.Unmarshal([]byte(message), data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerMemberUpdateFunctions {
			i(ctx, data.ServerId, data.Clear, data.Payload)
		}
	case "ServerMemberJoin":
		// Member join event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerMemberJoinFunctions {
			i(ctx, data.ID, data.User)
		}
	case "ServerMemberLeave":
		// Member left event.
		var data junk

		if err := json.Unmarshal([]byte(message), &data); err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerMemberLeaveFunctions {
			i(ctx, data.ID, data.User)
		}
	default:
		ln.Log(ctx, ln.F{"unknown_event": rawData.Type})
		// Unknown event.
		if c.OnUnknownEventFunctions != nil {
			for _, i := range c.OnUnknownEventFunctions {
				i(ctx, message)
			}
		}
	}
}

func (c *Client) handleCache(data string) {
	cache := &Cache{}

	err := json.Unmarshal([]byte(data), cache)

	if err != nil {
		fmt.Printf("Unexcepted Error: %s", err)
	}

	// Add client to users.
	for _, i := range cache.Users {
		i.Client = c
	}

	for _, i := range cache.Servers {
		i.Client = c
	}

	for _, i := range cache.Channels {
		i.Client = c
	}

	c.Cache = cache
}
