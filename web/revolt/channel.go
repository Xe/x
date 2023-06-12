package revolt

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/oklog/ulid/v2"
)

// Channel struct.
type Channel struct {
	Client    *Client
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
	DefaultPermissions uint        `json:"default_permissions"`
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

// Send a message to the channel.
func (c Channel) SendMessage(message *SendMessage) (*Message, error) {
	if message.Nonce == "" {
		message.CreateNonce()
	}

	respMessage := &Message{}
	respMessage.Client = c.Client
	msgData, err := json.Marshal(message)

	if err != nil {
		return respMessage, err
	}

	resp, err := c.Client.Request("POST", "/channels/"+c.Id+"/messages", msgData)

	if err != nil {
		return respMessage, err
	}

	err = json.Unmarshal(resp, respMessage)

	if err != nil {
		return respMessage, err
	}

	if message.DeleteAfter != 0 {
		go func() {
			time.Sleep(time.Second * time.Duration(message.DeleteAfter))
			respMessage.Delete()
		}()
	}

	return respMessage, nil
}

// Fetch messages from channel.
// Check: https://developers.revolt.chat/api/#tag/Messaging/paths/~1channels~1:channel~1messages/get for map parameters.
func (c Channel) FetchMessages(options map[string]interface{}) (*FetchedMessages, error) {
	// Format url
	url := "/channels/" + c.Id + "/messages?"

	for key, value := range options {
		if !reflect.ValueOf(value).IsZero() {
			url += fmt.Sprintf("%s=%v&", key, value)
		}
	}

	url = url[:len(url)-1]

	fetchedMsgs := &FetchedMessages{}

	// Send request
	resp, err := c.Client.Request("GET", url, []byte{})

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

	// Add client to users & messages
	for _, msg := range fetchedMsgs.Messages {
		msg.Client = c.Client
	}

	if fetchedMsgs.Users != nil {
		for _, msg := range fetchedMsgs.Users {
			msg.Client = c.Client
		}
	}

	return fetchedMsgs, nil
}

// Fetch a message from channel by Id.
func (c Channel) FetchMessage(id string) (*Message, error) {
	msg := &Message{}
	msg.Client = c.Client

	resp, err := c.Client.Request("GET", "/channels/"+c.Id+"/messages/"+id, []byte{})

	if err != nil {
		return msg, err
	}

	err = json.Unmarshal(resp, msg)
	return msg, err
}

// Edit channel.
func (c Channel) Edit(ec *EditChannel) error {
	data, err := json.Marshal(ec)

	if err != nil {
		return err
	}

	_, err = c.Client.Request("PATCH", "/channels/"+c.Id, data)
	return err
}

// Delete channel.
func (c Channel) Delete() error {
	_, err := c.Client.Request("DELETE", "/channels/"+c.Id, []byte{})
	return err
}

// Create a new invite.
// Returns a string (invite code) and error (nil if not exists).
func (c Channel) CreateInvite() (string, error) {
	data, err := c.Client.Request("POST", "/channels/"+c.Id+"/invites", []byte{})

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
func (c Channel) SetPermissions(role_id string, permissions uint) error {
	if role_id == "" {
		role_id = "default"
	}

	_, err := c.Client.Request("PUT", "/channels/"+c.Id+"/permissions/"+role_id, []byte(fmt.Sprintf("{\"permissions\":%d}", permissions)))
	return err
}

// Send a typing start event to the channel.
func (c *Channel) BeginTyping() {
	c.Client.Socket.SendText(fmt.Sprintf("{\"type\":\"BeginTyping\",\"channel\":\"%s\"}", c.Id))
}

// End the typing event in the channel.
func (c *Channel) EndTyping() {
	c.Client.Socket.SendText(fmt.Sprintf("{\"type\":\"EndTyping\",\"channel\":\"%s\"}", c.Id))
}
