package revolt

import (
	"context"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// Server struct.
type Server struct {
	CreatedAt time.Time

	Id                 string                 `json:"_id"`
	Nonce              string                 `json:"nonce"`
	OwnerId            string                 `json:"owner"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	ChannelIds         []string               `json:"channels"`
	Categories         []*ServerCategory      `json:"categories"`
	SystemMessages     *ServerSystemMessages  `json:"system_messages"`
	Roles              map[string]interface{} `json:"roles"`
	DefaultPermissions uint                   `json:"default_permissions"`
	Icon               *Attachment            `json:"icon"`
	Banner             *Attachment            `json:"banner"`
}

type Role struct {
	Name        string         `json:"name"`
	Permissions map[string]int `json:"permissions"`
	Color       string         `json:"colour"`
	Hoist       bool           `json:"hoist"`
	Rank        int            `json:"int"`
}

// Server categories struct.
type ServerCategory struct {
	Id         string   `json:"id"`
	Title      string   `json:"title"`
	ChannelIds []string `json:"channels"`
}

// System messages struct.
type ServerSystemMessages struct {
	UserJoined  string `json:"user_joined,omitempty"`
	UserLeft    string `json:"user_left,omitempty"`
	UserKicked  string `json:"user_kicked,omitempty"`
	UserBanned  string `json:"user_banned,omitempty"`
	UserTimeout string `json:"user_timeout,omitempty"`
}

// Server member struct.
type Member struct {
	Informations struct {
		ServerId string `json:"server"`
		UserId   string `json:"user"`
	} `json:"_id"`
	Nickname string      `json:"nickname"`
	Avatar   *Attachment `json:"avatar"`
	Roles    []string    `json:"roles"`
}

// Fetched server members struct.
type FetchedMembers struct {
	Members []*Member `json:"members"`
	Users   []*User   `json:"users"`
}

// Fetched bans struct.
type FetchedBans struct {
	Users []*User `json:"users"`
	Bans  []struct {
		Ids struct {
			UserId   string `json:"user"`
			ServerUd string `json:"server"`
		} `json:"_id"`
		Reason string `json:"reason"`
	} `json:"bans"`
}

// Calculate creation date and edit the struct.
func (s *Server) CalculateCreationDate() error {
	ulid, err := ulid.Parse(s.Id)

	if err != nil {
		return err
	}

	s.CreatedAt = time.UnixMilli(int64(ulid.Time()))
	return nil
}

// Edit server.
func (c *Client) ServerEdit(ctx context.Context, serverID string, es *EditServer) error {
	data, err := json.Marshal(es)

	if err != nil {
		return err
	}

	_, err = c.Request(ctx, "PATCH", "/servers/"+serverID, data)

	if err != nil {
		return err
	}

	return nil
}

// Delete / leave server.
// If the server not created by client, it will leave.
// Otherwise it will be deleted.
func (c *Client) ServerDelete(ctx context.Context, serverID string) error {
	_, err := c.Request(ctx, "DELETE", "/servers/"+serverID, []byte{})

	if err != nil {
		return err
	}

	return nil
}

// Create a new text-channel.
func (c *Client) CreateTextChannel(ctx context.Context, serverID, name, description string) (*Channel, error) {
	channel := &Channel{}

	data, err := c.Request(ctx, "POST", "/servers/"+serverID+"/channels", []byte("{\"type\":\"Text\",\"name\":\""+name+"\",\"description\":\""+description+"\",\"nonce\":\""+genULID()+"\"}"))

	if err != nil {
		return channel, err
	}

	err = json.Unmarshal(data, channel)

	if err != nil {
		return channel, err
	}

	return channel, nil
}

// Create a new voice-channel.
func (c *Client) CreateVoiceChannel(ctx context.Context, serverID, name, description string) (*Channel, error) {
	channel := &Channel{}

	data, err := c.Request(ctx, "POST", "/servers/"+serverID+"/channels", []byte("{\"type\":\"Voice\",\"name\":\""+name+"\",\"description\":\""+description+"\",\"nonce\":\""+genULID()+"\"}"))

	if err != nil {
		return channel, err
	}

	err = json.Unmarshal(data, channel)

	if err != nil {
		return channel, err
	}

	return channel, nil
}

// Fetch a member from Server.
func (c *Client) ServerFetchMember(ctx context.Context, serverID, id string) (*Member, error) {
	member := &Member{}

	data, err := c.Request(ctx, "GET", "/servers/"+serverID+"/members/"+id, []byte{})

	if err != nil {
		return member, err
	}

	err = json.Unmarshal(data, member)

	if err != nil {
		return member, err
	}

	return member, nil
}

// Fetch all of the members from Server.
func (c *Client) ServerFetchMembers(ctx context.Context, serverID string) (*FetchedMembers, error) {
	members := &FetchedMembers{}

	data, err := c.Request(ctx, "GET", "/servers/"+serverID+"/members", []byte{})

	if err != nil {
		return members, err
	}

	err = json.Unmarshal(data, members)

	if err != nil {
		return members, err
	}

	return members, nil
}

// Edit a member.
func (c *Client) ServerEditMember(ctx context.Context, serverID, id string, em *EditMember) error {
	data, err := json.Marshal(em)

	if err != nil {
		return err
	}

	_, err = c.Request(ctx, "PATCH", "/servers/"+serverID+"/members/"+id, data)

	if err != nil {
		return err
	}

	return nil
}

// Kick a member from server.
func (c *Client) ServerKickMember(ctx context.Context, serverID, id string) error {
	_, err := c.Request(ctx, "DELETE", "/servers/"+serverID+"/members/"+id, []byte{})

	if err != nil {
		return err
	}

	return nil
}

// Ban a member from server.
func (c *Client) ServerBanMember(ctx context.Context, serverID, id, reason string) error {
	data, err := json.Marshal(map[string]string{"reason": reason})
	if err != nil {
		return err
	}
	_, err = c.Request(ctx, "PUT", "/servers/"+serverID+"/bans/"+id, data)

	if err != nil {
		return err
	}

	return nil
}

// Unban a member from server.
func (c *Client) ServerUnbanMember(ctx context.Context, serverID, id string) error {
	_, err := c.Request(ctx, "DELETE", "/servers/"+serverID+"/bans/"+id, []byte{})
	if err != nil {
		return err
	}

	return nil
}

// Fetch server bans.
func (c *Client) ServerFetchBans(ctx context.Context, serverID string) (*FetchedBans, error) {
	bans := &FetchedBans{}

	data, err := c.Request(ctx, "GET", "/servers/"+serverID+"/bans", []byte{})
	if err != nil {
		return bans, err
	}

	err = json.Unmarshal(data, bans)
	if err != nil {
		return bans, err
	}

	return bans, nil
}

// Timeout a member from server. Placeholder.
func (c *Client) ServerTimeoutMember(ctx context.Context, serverID, id string) error {
	return nil
}

// Set server permissions for a role.
// Leave role field empty if you want to edit default permissions
func (c *Client) ServerSetRolePermissions(ctx context.Context, serverID, roleID string, channelPermissions, serverPermissions uint) error {
	data, err := json.Marshal(map[string]any{
		"permissions": map[string]any{
			"server":  serverPermissions,
			"channel": channelPermissions,
		},
	})
	if roleID == "" {
		roleID = "default"
	}

	_, err = c.Request(ctx, "PUT", "/servers/"+serverID+"/permissions/"+roleID, data)

	if err != nil {
		return err
	}

	return nil
}

// Create a new role for server.
// Returns string (role id) and error.
func (c *Client) ServerCreateRole(ctx context.Context, serverID, name string) (string, error) {
	role := &struct {
		ID          string `json:"id"`
		Permissions []uint `json:"permissions"`
	}{}

	data, err := json.Marshal(map[string]string{
		"name": name,
	})
	if err != nil {
		return role.ID, err
	}

	data, err = c.Request(ctx, "POST", "/servers/"+serverID+"/roles", data)
	if err != nil {
		return role.ID, err
	}

	err = json.Unmarshal(data, role)
	if err != nil {
		return role.ID, err
	}

	return role.ID, nil
}

// Edit a server role.
func (c *Client) ServerEditRole(ctx context.Context, serverID, id string, er *EditRole) error {
	data, err := json.Marshal(er)
	if err != nil {
		return err
	}

	_, err = c.Request(ctx, "PATCH", "/servers/"+serverID+"/roles/"+id, data)
	if err != nil {
		return err
	}

	return nil
}

// Delete a server role.
func (c *Client) ServerDeleteRole(ctx context.Context, serverID, id string) error {
	_, err := c.Request(ctx, "DELETE", "/servers/"+serverID+"/roles/"+id, []byte{})
	if err != nil {
		return err
	}

	return nil
}

// Fetch server invite.
func (c *Client) ServerFetchInvites(ctx context.Context, serverID string) error {
	_, err := c.Request(ctx, "GET", "/servers/"+serverID+"/invites", []byte{})
	if err != nil {
		return err
	}

	return nil
}

// Mark a server as read.
func (c *Client) MarkServerAsRead(ctx context.Context, id string) error {
	_, err := c.Request(ctx, "PUT", "/servers/"+id+"/ack", []byte{})

	if err != nil {
		return err
	}

	return nil
}
