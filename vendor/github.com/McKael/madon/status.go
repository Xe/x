/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
)

// updateStatusOptions contains option fields for POST and DELETE API calls
type updateStatusOptions struct {
	// The ID is used for most commands
	ID int64

	// The following fields are used for posting a new status
	Status      string
	InReplyToID int64
	MediaIDs    []int64
	Sensitive   bool
	SpoilerText string
	Visibility  string // "direct", "private", "unlisted" or "public"
}

// getMultipleStatuses returns a list of status entities
// If lopt.All is true, several requests will be made until the API server
// has nothing to return.
func (mc *Client) getMultipleStatuses(endPoint string, params apiCallParams, lopt *LimitParams) ([]Status, error) {
	var statuses []Status
	var links apiLinks
	if err := mc.apiCall(endPoint, rest.Get, params, lopt, &links, &statuses); err != nil {
		return nil, err
	}
	if lopt != nil { // Fetch more pages to reach our limit
		var statusSlice []Status
		for (lopt.All || lopt.Limit > len(statuses)) && links.next != nil {
			newlopt := links.next
			links = apiLinks{}
			if err := mc.apiCall(endPoint, rest.Get, params, newlopt, &links, &statusSlice); err != nil {
				return nil, err
			}
			statuses = append(statuses, statusSlice...)
			statusSlice = statusSlice[:0] // Clear struct
		}
	}
	return statuses, nil
}

// queryStatusData queries the statuses API
// The operation 'op' can be empty or "status" (the status itself), "context",
// "card", "reblogged_by", "favourited_by".
// The data argument will receive the object(s) returned by the API server.
func (mc *Client) queryStatusData(statusID int64, op string, data interface{}) error {
	if statusID < 1 {
		return ErrInvalidID
	}

	endPoint := "statuses/" + strconv.FormatInt(statusID, 10)

	if op != "" && op != "status" {
		switch op {
		case "context", "card", "reblogged_by", "favourited_by":
		default:
			return ErrInvalidParameter
		}

		endPoint += "/" + op
	}

	return mc.apiCall(endPoint, rest.Get, nil, nil, nil, data)
}

// updateStatusData updates the statuses
// The operation 'op' can be empty or "status" (to post a status), "delete"
// (for deleting a status), "reblog"/"unreblog", "favourite"/"unfavourite",
// "mute"/"unmute" (for conversations) or "pin"/"unpin".
// The data argument will receive the object(s) returned by the API server.
func (mc *Client) updateStatusData(op string, opts updateStatusOptions, data interface{}) error {
	method := rest.Post
	endPoint := "statuses"
	params := make(apiCallParams)

	switch op {
	case "", "status":
		op = "status"
		if opts.Status == "" {
			return ErrInvalidParameter
		}
		switch opts.Visibility {
		case "", "direct", "private", "unlisted", "public":
			// Okay
		default:
			return ErrInvalidParameter
		}
		if len(opts.MediaIDs) > 4 {
			return errors.New("too many (>4) media IDs")
		}
	case "delete":
		method = rest.Delete
		if opts.ID < 1 {
			return ErrInvalidID
		}
		endPoint += "/" + strconv.FormatInt(opts.ID, 10)
	case "reblog", "unreblog", "favourite", "unfavourite":
		if opts.ID < 1 {
			return ErrInvalidID
		}
		endPoint += "/" + strconv.FormatInt(opts.ID, 10) + "/" + op
	case "mute", "unmute", "pin", "unpin":
		if opts.ID < 1 {
			return ErrInvalidID
		}
		endPoint += "/" + strconv.FormatInt(opts.ID, 10) + "/" + op
	default:
		return ErrInvalidParameter
	}

	// Form items for a new toot
	if op == "status" {
		params["status"] = opts.Status
		if opts.InReplyToID > 0 {
			params["in_reply_to_id"] = strconv.FormatInt(opts.InReplyToID, 10)
		}
		for i, id := range opts.MediaIDs {
			if id < 1 {
				return ErrInvalidID
			}
			qID := fmt.Sprintf("media_ids[%d]", i+1)
			params[qID] = strconv.FormatInt(id, 10)
		}
		if opts.Sensitive {
			params["sensitive"] = "true"
		}
		if opts.SpoilerText != "" {
			params["spoiler_text"] = opts.SpoilerText
		}
		if opts.Visibility != "" {
			params["visibility"] = opts.Visibility
		}
	}

	return mc.apiCall(endPoint, method, params, nil, nil, data)
}

// GetStatus returns a status
// The returned status can be nil if there is an error or if the
// requested ID does not exist.
func (mc *Client) GetStatus(statusID int64) (*Status, error) {
	var status Status

	if err := mc.queryStatusData(statusID, "status", &status); err != nil {
		return nil, err
	}
	if status.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return &status, nil
}

// GetStatusContext returns a status context
func (mc *Client) GetStatusContext(statusID int64) (*Context, error) {
	var context Context
	if err := mc.queryStatusData(statusID, "context", &context); err != nil {
		return nil, err
	}
	return &context, nil
}

// GetStatusCard returns a status card
func (mc *Client) GetStatusCard(statusID int64) (*Card, error) {
	var card Card
	if err := mc.queryStatusData(statusID, "card", &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// GetStatusRebloggedBy returns a list of the accounts who reblogged a status
func (mc *Client) GetStatusRebloggedBy(statusID int64, lopt *LimitParams) ([]Account, error) {
	o := &getAccountsOptions{ID: statusID, Limit: lopt}
	return mc.getMultipleAccountsHelper("reblogged_by", o)
}

// GetStatusFavouritedBy returns a list of the accounts who favourited a status
func (mc *Client) GetStatusFavouritedBy(statusID int64, lopt *LimitParams) ([]Account, error) {
	o := &getAccountsOptions{ID: statusID, Limit: lopt}
	return mc.getMultipleAccountsHelper("favourited_by", o)
}

// PostStatus posts a new "toot"
// All parameters but "text" can be empty.
// Visibility must be empty, or one of "direct", "private", "unlisted" and "public".
func (mc *Client) PostStatus(text string, inReplyTo int64, mediaIDs []int64, sensitive bool, spoilerText string, visibility string) (*Status, error) {
	var status Status
	o := updateStatusOptions{
		Status:      text,
		InReplyToID: inReplyTo,
		MediaIDs:    mediaIDs,
		Sensitive:   sensitive,
		SpoilerText: spoilerText,
		Visibility:  visibility,
	}

	err := mc.updateStatusData("status", o, &status)
	if err != nil {
		return nil, err
	}
	if status.ID == 0 {
		return nil, ErrEntityNotFound // TODO Change error message
	}
	return &status, err
}

// DeleteStatus deletes a status
func (mc *Client) DeleteStatus(statusID int64) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("delete", o, &status)
	return err
}

// ReblogStatus reblogs a status
func (mc *Client) ReblogStatus(statusID int64) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("reblog", o, &status)
	return err
}

// UnreblogStatus unreblogs a status
func (mc *Client) UnreblogStatus(statusID int64) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("unreblog", o, &status)
	return err
}

// FavouriteStatus favourites a status
func (mc *Client) FavouriteStatus(statusID int64) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("favourite", o, &status)
	return err
}

// UnfavouriteStatus unfavourites a status
func (mc *Client) UnfavouriteStatus(statusID int64) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("unfavourite", o, &status)
	return err
}

// PinStatus pins a status
func (mc *Client) PinStatus(statusID int64) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("pin", o, &status)
	return err
}

// UnpinStatus unpins a status
func (mc *Client) UnpinStatus(statusID int64) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("unpin", o, &status)
	return err
}

// MuteConversation mutes the conversation containing a status
func (mc *Client) MuteConversation(statusID int64) (*Status, error) {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("mute", o, &status)
	return &status, err
}

// UnmuteConversation unmutes the conversation containing a status
func (mc *Client) UnmuteConversation(statusID int64) (*Status, error) {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("unmute", o, &status)
	return &status, err
}
