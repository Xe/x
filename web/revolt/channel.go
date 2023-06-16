package revolt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/oklog/ulid/v2"
)

// Channel struct.
type Channel struct {
	CreatedAt time.Time

	Id                 string      `json:"_id"`
	Nonce              string      `json:"nonce"`
	OwnerId            string      `json:"owner"`
	Name               string      `json:"name"`
	Active             bool        `json:"active"`
	Recipients         []string    `json:"recipients"`
	LastMessage        interface{} `json:"last_message"`
	Description        string      `json:"description"`
	Icon               *Attachment `json:"icon"`
	DefaultPermissions interface{} `json:"default_permissions"`
	RolePermissions    interface{} `json:"role_permissions"`
	Permissions        uint        `json:"permissions"`
}

// Fetched messages struct.
type FetchedMessages struct {
	Messages []*Message `json:"messages"`
	Users    []*User    `json:"users"`
}

// Calculate creation date and edit the struct.
func (c *Channel) CalculateCreationDate() error {
	ulid, err := ulid.Parse(c.Id)

	if err != nil {
		return err
	}

	c.CreatedAt = time.UnixMilli(int64(ulid.Time()))
	return nil
}

// SendMessage sends a message to a channel.
func (c *Client) ChannelSendMessage(ctx context.Context, channelID string, message *SendMessage) (*Message, error) {
	data, err := json.Marshal(message)

	if err != nil {
		return nil, err
	}

	resp, err := c.Request(ctx, "POST", "/channels/"+channelID+"/messages", data)
	if err != nil {
		return nil, err
	}

	msg := &Message{}
	err = json.Unmarshal(resp, msg)

	if err != nil {
		return nil, err
	}

	if message.DeleteAfter != 0 {
		go func() {
			time.Sleep(time.Second * time.Duration(message.DeleteAfter))
			c.MessageDelete(ctx, channelID, msg.ID)
		}()
	}

	return msg, nil
}

// Fetch messages from channel.
// Check: https://developers.revolt.chat/api/#tag/Messaging/paths/~1channels~1:channel~1messages/get for map parameters.
func (c *Client) ChannelFetchMessages(ctx context.Context, channelID string, options url.Values) (*FetchedMessages, error) {
	// Format url
	url := "/channels/" + channelID + "/messages?" + options.Encode()

	fetchedMsgs := &FetchedMessages{}

	// Send request
	resp, err := c.Request(ctx, "GET", url, []byte{})

	if err != nil {
		return fetchedMsgs, err
	}

	err = json.Unmarshal(resp, &fetchedMsgs)

	if err != nil {
		err = json.Unmarshal([]byte(fmt.Sprintf("{\"messages\": %s}", resp)), &fetchedMsgs)

		if err != nil {
			return fetchedMsgs, err
		}
	}

	return fetchedMsgs, nil
}

// Fetch a message from channel by Id.
func (c *Client) ChannelFetchMessage(ctx context.Context, channelID, id string) (*Message, error) {
	msg := &Message{}

	resp, err := c.Request(ctx, "GET", "/channels/"+channelID+"/messages/"+id, []byte{})

	if err != nil {
		return msg, err
	}

	err = json.Unmarshal(resp, msg)
	return msg, err
}

// Edit channel.
func (c *Client) ChannelEdit(ctx context.Context, channelID string, ec *EditChannel) error {
	data, err := json.Marshal(ec)
	if err != nil {
		return err
	}

	_, err = c.Request(ctx, "PATCH", "/channels/"+channelID, data)
	return err
}

// Delete channel.
func (c *Client) ChannelDelete(ctx context.Context, channelID string) error {
	_, err := c.Request(ctx, "DELETE", "/channels/"+channelID, []byte{})
	return err
}

// Create a new invite.
// Returns a string (invite code) and error (nil if not exists).
func (c *Client) ChannelCreateInvite(ctx context.Context, channelID string) (string, error) {
	data, err := c.Request(ctx, "POST", "/channels/"+channelID+"/invites", []byte{})

	if err != nil {
		return "", err
	}

	dataStruct := &struct {
		InviteCode string `json:"code"`
	}{}

	err = json.Unmarshal(data, dataStruct)
	return dataStruct.InviteCode, err
}

// Set channel permissions for a role.
// Leave role field empty if you want to edit default permissions
func (c *Client) ChannelSetPermissions(ctx context.Context, channelID, roleID string, permissions uint) error {
	if roleID == "" {
		roleID = "default"
	}

	data, err := json.Marshal(struct {
		Permissions uint `json:"permissions"`
	}{Permissions: permissions})
	if err != nil {
		return err
	}

	_, err = c.Request(ctx, "PUT", "/channels/"+channelID+"/permissions/"+roleID, data)
	return err
}

// // Send a typing start event to the channel.
// func (c *Channel) BeginTyping() {
// 	c.Client.Socket.SendText(fmt.Sprintf("{\"type\":\"BeginTyping\",\"channel\":\"%s\"}", c.Id))
// }

// // End the typing event in the channel.
// func (c *Channel) EndTyping() {
// 	c.Client.Socket.SendText(fmt.Sprintf("{\"type\":\"EndTyping\",\"channel\":\"%s\"}", c.Id))
// }
