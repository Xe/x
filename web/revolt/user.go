package revolt

import (
	"context"
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
func (c *Client) UserOpenDirectMessage(ctx context.Context, uid string) (*Channel, error) {
	dmChannel := &Channel{}

	resp, err := c.Request(ctx, "GET", "/users/"+uid+"/dm", []byte{})

	if err != nil {
		return dmChannel, err
	}

	err = json.Unmarshal(resp, dmChannel)
	return dmChannel, err
}

// Fetch default user avatar.
func (c *Client) UserFetchDefaultAvatar(ctx context.Context, uid string) (*Binary, error) {
	avatarData := &Binary{}

	resp, err := c.Request(ctx, "GET", "/users/"+uid+"/default_avatar", []byte{})

	if err != nil {
		return avatarData, err
	}

	avatarData.Data = resp
	return avatarData, nil
}

// Fetch user relationship.
func (c *Client) UserFetchRelationship(ctx context.Context, uid string) (*UserRelations, error) {
	relationshipData := &UserRelations{}
	relationshipData.Id = uid

	resp, err := c.Request(ctx, "GET", "/users/"+uid+"/relationship", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}

// Block user.
func (c *Client) UserBlock(ctx context.Context, uid string) (*UserRelations, error) {
	relationshipData := &UserRelations{}
	relationshipData.Id = uid

	resp, err := c.Request(ctx, "PUT", "/users/"+uid+"/block", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}

// Un-block user.
func (c *Client) UserUnblock(ctx context.Context, uid string) (*UserRelations, error) {
	relationshipData := &UserRelations{}
	relationshipData.Id = uid

	resp, err := c.Request(ctx, "DELETE", "/users/"+uid+"/block", []byte{})

	if err != nil {
		return relationshipData, err
	}

	err = json.Unmarshal(resp, relationshipData)
	return relationshipData, err
}
