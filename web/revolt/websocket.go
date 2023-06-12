package revolt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sacOO7/gowebsocket"
)

func (c *Client) Start() {
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
			c.Destroy()
			panic(err)
		}

		if rawData.Type == "Authenticated" {
			go c.ping()
		}

		// Handle events.
		c.handleEvents(rawData, message)
		// fmt.Println(message)
	}

	// Start connection.
	c.Socket.Connect()
}

// Handle on connected.
func (c *Client) handleWebsocketAuth() {
	if c.SelfBot == nil {
		c.Socket.SendText(fmt.Sprintf("{\"type\":\"Authenticate\",\"token\":\"%s\"}", c.Token))
	} else {
		c.Socket.SendText(fmt.Sprintf("{\"type\":\"Authenticate\",\"result\":\"Success\",\"_id\":\"%s\",\"token\":\"%s\",\"user_id\":\"%s\",\"name\":\"revolt\"}", c.SelfBot.Id, c.SelfBot.SessionToken, c.SelfBot.UserId))
	}
}

// Destroy the websocket.
func (c *Client) Destroy() {
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
func (c *Client) handleEvents(rawData *struct {
	Type string `json:"type"`
}, message string) {
	if rawData.Type == "Ready" {
		// Add cache.
		c.handleCache(message)

		// onReady event
		if c.OnReadyFunctions != nil {
			for _, i := range c.OnReadyFunctions {
				i()
			}
		}
	} else if rawData.Type == "Message" && c.OnMessageFunctions != nil {
		// Message create event.
		msgData := &Message{}
		msgData.Client = c

		err := json.Unmarshal([]byte(message), msgData)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnMessageFunctions {
			i(msgData)
		}
	} else if rawData.Type == "MessageUpdate" && c.OnMessageUpdateFunctions != nil {
		// Message update event.
		data := &struct {
			ChannelId string                 `json:"channel"`
			MessageId string                 `json:"id"`
			Payload   map[string]interface{} `json:"data"`
		}{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnMessageUpdateFunctions {
			i(data.ChannelId, data.MessageId, data.Payload)
		}
	} else if rawData.Type == "MessageDelete" && c.OnMessageDeleteFunctions != nil {
		// Message delete event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnMessageDeleteFunctions {
			i((*data)["channel"], (*data)["id"])
		}
	} else if rawData.Type == "ChannelCreate" && c.OnChannelCreateFunctions != nil {
		// Channel create event.
		channelData := &Channel{}
		channelData.Client = c

		err := json.Unmarshal([]byte(message), channelData)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelCreateFunctions {
			i(channelData)
		}
	} else if rawData.Type == "ChannelUpdate" && c.OnChannelUpdateFunctions != nil {
		// Channel update event.
		data := &struct {
			ChannelId string                 `json:"id"`
			Clear     string                 `json:"clear"`
			Payload   map[string]interface{} `json:"data"`
		}{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelUpdateFunctions {
			i(data.ChannelId, data.Clear, data.Payload)
		}
	} else if rawData.Type == "ChannelDelete" && c.OnChannelDeleteFunctions != nil {
		// Channel delete event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelDeleteFunctions {
			i((*data)["id"])
		}
	} else if rawData.Type == "GroupCreate" && c.OnGroupCreateFunctions != nil {
		// Group channel create event.
		groupChannelData := &Group{}
		groupChannelData.Client = c
	
		err := json.Unmarshal([]byte(message), groupChannelData)
	
		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}
	
		for _, i := range c.OnGroupCreateFunctions {
			i(groupChannelData)
		}
	} else if rawData.Type == "GroupMemberAdded" && c.OnGroupMemberAddedFunctions != nil {
		// Group member added event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnGroupMemberAddedFunctions {
			i((*data)["id"], (*data)["user"])
		}
	} else if rawData.Type == "GroupMemberRemoved" && c.OnGroupMemberRemovedFunctions != nil {
		// Group member removed event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnGroupMemberRemovedFunctions {
			i((*data)["id"], (*data)["user"])
		}
	} else if rawData.Type == "ChannelStartTyping" && c.OnChannelStartTypingFunctions != nil {
		// Channel start typing event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelStartTypingFunctions {
			i((*data)["id"], (*data)["user"])
		}
	} else if rawData.Type == "ChannelStopTyping" && c.OnChannelStopTypingFunctions != nil {
		// Channel stop typing event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnChannelStopTypingFunctions {
			i((*data)["id"], (*data)["user"])
		}
	} else if rawData.Type == "ServerCreate" && c.OnServerCreateFunctions != nil {
		// Server create event.
		serverData := &Server{}
		serverData.Client = c
	
		err := json.Unmarshal([]byte(message), serverData)
	
		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}
	
		for _, i := range c.OnServerCreateFunctions {
			i(serverData)
		}
	} else if rawData.Type == "ServerUpdate" && c.OnServerUpdateFunctions != nil {
		// Server update event.
		data := &struct {
			ServerId string                 `json:"id"`
			Clear    string                 `json:"clear"`
			Payload  map[string]interface{} `json:"data"`
		}{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerUpdateFunctions {
			i(data.ServerId, data.Clear, data.Payload)
		}
	} else if rawData.Type == "ServerDelete" && c.OnServerDeleteFunctions != nil {
		// Server delete event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerDeleteFunctions {
			i((*data)["id"])
		}
	} else if rawData.Type == "ServerMemberUpdate" && c.OnServerMemberUpdateFunctions != nil {
		// Member update event.
		data := &struct {
			ServerId string                 `json:"id"`
			Clear    string                 `json:"clear"`
			Payload  map[string]interface{} `json:"data"`
		}{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerMemberUpdateFunctions {
			i(data.ServerId, data.Clear, data.Payload)
		}
	} else if rawData.Type == "ServerMemberJoin" && c.OnServerMemberJoinFunctions != nil {
		// Member join event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerMemberJoinFunctions {
			i((*data)["id"], (*data)["user"])
		}
	} else if rawData.Type == "ServerMemberLeave" && c.OnServerMemberLeaveFunctions != nil {
		// Member left event.
		data := &map[string]string{}

		err := json.Unmarshal([]byte(message), data)

		if err != nil {
			fmt.Printf("Unexcepted Error: %s", err)
		}

		for _, i := range c.OnServerMemberLeaveFunctions {
			i((*data)["id"], (*data)["user"])
		}
	} else {
		// Unknown event.
		if c.OnUnknownEventFunctions != nil {
			for _, i := range c.OnUnknownEventFunctions {
				i(message)
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
