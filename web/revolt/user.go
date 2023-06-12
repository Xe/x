package revolt

import (
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// User struct.
type User struct {
	Client    *Client
	CreatedAt time.Time

	Id             string           `json:"_id"`
	Username       string           `json:"username"`
	Avatar         *Attachment      `json:"avatar"`
	Relations      []*UserRelations `json:"relations"`
	Badges         int              `json:"badges"`
	Status         *UserStatus      `json:"status"`
	Relationship   string           `json:"relationship"`
	IsOnline       bool             `json:"online"`
	Flags          int              `json:"flags"`
	BotInformation *BotInformation  `json:"bot"`
}

// User relations struct.
type UserRelations struct {
	Id     string `json:"_id"`
	Status string `json:"status"`
}

// User status struct.
type UserStatus struct {
	Text     string `json:"text"`
	Presence string `json:"presence"`
}

// Bot information struct.
type BotInformation struct {
	Owner string `json:"owner"`
}

// Calculate creation date and edit the struct.
func (u *User) CalculateCreationDate() error {
	ulid, err := ulid.Parse(u.Id)

	if err != nil {
		return err
	}

	u.CreatedAt = time.UnixMilli(int64(ulid.Time()))
	return nil
}

// Create a mention format.
func (u User) FormatMention() string {
	return "<@" + u.Id + ">"
}

// Open a DM with the user.
func (u User) OpenDirectMessage() (*Channel, error) {
	dmChannel := &Channel{}
	dmChannel.Client = u.Client

	resp, err := u.Client.Request("GET", "/users/"+u.Id+"/dm", []byte{})

	if err != nil {
		return dmChannel, err
	}

	err = json.Unmarshal(resp, dmChannel)
	return dmChannel, err
}

// Fetch default user avatar.
func (u User) FetchDefaultAvatar() (*Binary, error) {
	avatarData := &Binary{}

	resp, err := u.Client.Request("GET", "/users/"+u.Id+"/default_avatar", []byte{})

	if err != nil {
		return avatarData, err
	}

	avatarData.Data = resp
	return avatarData, nil
}

// Fetch user relationship.
func (u User) FetchRelationship() (*UserRelations, error) {
	relationshipData := &UserRelations{}
	relationshipData.Id = u.Id

	resp, err := u.Client.Request("GET", "/users/"+u.Id+"/relationship", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}

// Block user.
func (u User) Block() (*UserRelations, error) {
	relationshipData := &UserRelations{}
	relationshipData.Id = u.Id

	resp, err := u.Client.Request("PUT", "/users/"+u.Id+"/block", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}

// Un-block user.
func (u User) Unblock() (*UserRelations, error) {
	relationshipData := &UserRelations{}
	relationshipData.Id = u.Id

	resp, err := u.Client.Request("DELETE", "/users/"+u.Id+"/block", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}
