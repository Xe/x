package revolt

import (
	"context"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// Group channel struct.
type Group struct {
	CreatedAt time.Time

	Id          string   `json:"_id"`
	Nonce       string   `json:"nonce"`
	OwnerId     string   `json:"owner"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Users       []string `json:"users"`
}

// Fetched group members struct.
type FetchedGroupMembers struct {
	Messages []*Message `json:"messages"`
	Users    []*User    `json:"users"`
}

// System messages struct.
type GroupSystemMessages struct {
	UserJoined string `json:"user_joined,omitempty"`
	UserLeft   string `json:"user_left,omitempty"`
}

// Calculate creation date and edit the struct.
func (c *Group) CalculateCreationDate() error {
	ulid, err := ulid.Parse(c.Id)

	if err != nil {
		return err
	}

	c.CreatedAt = time.UnixMilli(int64(ulid.Time()))
	return nil
}

// Fetch all of the members from group.
func (c *Client) GroupFetchMembers(ctx context.Context, groupID string) ([]*User, error) {
	var groupMembers []*User

	resp, err := c.Request(ctx, "GET", "/channels/"+groupID+"/members", []byte{})

	if err != nil {
		return groupMembers, err
	}

	err = json.Unmarshal(resp, &groupMembers)
	return groupMembers, err
}

// Add a new group recipient.
func (c *Client) AddGroupRecipient(ctx context.Context, groupID, userID string) error {
	_, err := c.Request(ctx, "PUT", "/channels/"+groupID+"/recipients/"+userID, []byte{})
	return err
}

// Delete a group recipient.
func (c *Client) DeleteGroupRecipient(ctx context.Context, groupID, userID string) error {
	_, err := c.Request(ctx, "DELETE", "/channels/"+groupID+"/recipients/"+userID, []byte{})
	return err
}
