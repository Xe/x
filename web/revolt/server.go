package revolt

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// Server struct.
type Server struct {
	Client    *Client
	CreatedAt time.Time

	Id                 string                       `json:"_id"`
	Nonce              string                       `json:"nonce"`
	OwnerId            string                       `json:"owner"`
	Name               string                       `json:"name"`
	Description        string                       `json:"description"`
	ChannelIds         []string                     `json:"channels"`
	Categories         []*ServerCategory            `json:"categories"`
	SystemMessages     *ServerSystemMessages        `json:"system_messages"`
	Roles              map[string]interface{}       `json:"roles"`
	DefaultPermissions uint                         `json:"default_permissions"`
	Icon               *Attachment                  `json:"icon"`
	Banner             *Attachment                  `json:"banner"`
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
func (s Server) Edit(es *EditServer) error {
	data, err := json.Marshal(es)

	if err != nil {
		return err
	}

	_, err = s.Client.Request("PATCH", "/servers/"+s.Id, data)

	if err != nil {
		return err
	}

	return nil
}

// Delete / leave server.
// If the server not created by client, it will leave.
// Otherwise it will be deleted.
func (s Server) Delete() error {
	_, err := s.Client.Request("DELETE", "/servers/"+s.Id, []byte{})

	if err != nil {
		return err
	}

	return nil
}

// Create a new text-channel.
func (s Server) CreateTextChannel(name, description string) (*Channel, error) {
	channel := &Channel{}
	channel.Client = s.Client

	data, err := s.Client.Request("POST", "/servers/"+s.Id+"/channels", []byte("{\"type\":\"Text\",\"name\":\""+name+"\",\"description\":\""+description+"\",\"nonce\":\""+genULID()+"\"}"))

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
func (s Server) CreateVoiceChannel(name, description string) (*Channel, error) {
	channel := &Channel{}
	channel.Client = s.Client

	data, err := s.Client.Request("POST", "/servers/"+s.Id+"/channels", []byte("{\"type\":\"Voice\",\"name\":\""+name+"\",\"description\":\""+description+"\",\"nonce\":\""+genULID()+"\"}"))

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
func (s Server) FetchMember(id string) (*Member, error) {
	member := &Member{}

	data, err := s.Client.Request("GET", "/servers/"+s.Id+"/members/"+id, []byte{})

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
func (s Server) FetchMembers() (*FetchedMembers, error) {
	members := &FetchedMembers{}

	data, err := s.Client.Request("GET", "/servers/"+s.Id+"/members", []byte{})

	if err != nil {
		return members, err
	}

	err = json.Unmarshal(data, members)

	if err != nil {
		return members, err
	}

	// Add client to the user
	for _, i := range members.Users {
		i.Client = s.Client
	}

	return members, nil
}

// Edit a member.
func (s Server) EditMember(id string, em *EditMember) error {
	data, err := json.Marshal(em)

	if err != nil {
		return err
	}

	_, err = s.Client.Request("PATCH", "/servers/"+s.Id+"/members/"+id, data)

	if err != nil {
		return err
	}

	return nil
}

// Kick a member from server.
func (s Server) KickMember(id string) error {
	_, err := s.Client.Request("DELETE", "/servers/"+s.Id+"/members/"+id, []byte{})

	if err != nil {
		return err
	}

	return nil
}

// Ban a member from server.
func (s Server) BanMember(id, reason string) error {
	_, err := s.Client.Request("PUT", "/servers/"+s.Id+"/bans/"+id, []byte("{\"reason\":\""+reason+"\"}"))

	if err != nil {
		return err
	}

	return nil
}

// Unban a member from server.
func (s Server) UnbanMember(id string) error {
	_, err := s.Client.Request("DELETE", "/servers/"+s.Id+"/bans/"+id, []byte{})

	if err != nil {
		return err
	}

	return nil
}

// Fetch server bans.
func (s Server) FetchBans() (*FetchedBans, error) {
	bans := &FetchedBans{}

	data, err := s.Client.Request("GET", "/servers/"+s.Id+"/bans", []byte{})

	if err != nil {
		return bans, err
	}

	err = json.Unmarshal(data, bans)

	if err != nil {
		return bans, err
	}

	// Add client to the user
	for _, i := range bans.Users {
		i.Client = s.Client
	}

	return bans, nil
}

// Timeout a member from server.
func (s Server) TimeoutMember(id string) error {
	// Placeholder for timeout.

	return nil
}

// Set server permissions for a role.
// Leave role field empty if you want to edit default permissions
func (s Server) SetPermissions(role_id string, channel_permissions, server_permissions uint) error {
	if role_id == "" {
		role_id = "default"
	}

	_, err := s.Client.Request("PUT", "/servers/"+s.Id+"/permissions/"+role_id, []byte(fmt.Sprintf("{\"permissions\":{\"server\":%d,\"channel\":%d}}", channel_permissions, server_permissions)))

	if err != nil {
		return err
	}

	return nil
}

// Create a new role for server.
// Returns string (role id), uint (server perms), uint (channel perms) and error.
func (s Server) CreateRole(name string) (string, uint, uint, error) {
	role := &struct {
		Id          string `json:"id"`
		Permissions []uint `json:"permissions"`
	}{}

	data, err := s.Client.Request("POST", "/servers/"+s.Id+"/roles", []byte("{\"name\":\""+name+"\"}"))

	if err != nil {
		return role.Id, 0, 0, err
	}

	err = json.Unmarshal(data, role)

	if err != nil {
		return role.Id, 0, 0, err
	}

	return role.Id, role.Permissions[0], role.Permissions[1], nil
}

// Edit a server role.
func (s Server) EditRole(id string, er *EditRole) error {
	data, err := json.Marshal(er)

	if err != nil {
		return err
	}

	_, err = s.Client.Request("PATCH", "/servers/"+s.Id+"/roles/"+id, data)

	if err != nil {
		return err
	}

	return nil
}

// Delete a server role.
func (s Server) DeleteRole(id string) error {
	_, err := s.Client.Request("DELETE", "/servers/"+s.Id+"/roles/"+id, []byte{})

	if err != nil {
		return err
	}

	return nil
}

// Fetch server invite.
func (s Server) FetchInvites(id string) error {
	_, err := s.Client.Request("GET", "/servers/"+id+"/invites", []byte{})

	if err != nil {
		return err
	}

	return nil
}

// Mark a server as read.
func (s Server) MarkServerAsRead(id string) error {
	_, err := s.Client.Request("PUT", "/servers/"+id+"/ack", []byte{})

	if err != nil {
		return err
	}

	return nil
}
