/*
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"fmt"
	"strconv"

	"github.com/sendgrid/rest"
)

// updateStatusOptions contains option fields for POST and DELETE API calls
type updateStatusOptions struct {
	// The ID is used for most commands
	ID int

	// The following fields are used for posting a new status
	Status      string
	InReplyToID int
	MediaIDs    []int
	Sensitive   bool
	SpoilerText string
	Visibility  string // "direct", "private", "unlisted" or "public"
}

// queryStatusData queries the statuses API
// The operation 'op' can be empty or "status" (the status itself), "context",
// "card", "reblogged_by", "favourited_by".
// The data argument will receive the object(s) returned by the API server.
func (mc *Client) queryStatusData(statusID int, op string, data interface{}) error {
	if statusID < 1 {
		return ErrInvalidID
	}

	endPoint := "statuses/" + strconv.Itoa(statusID)

	if op != "" && op != "status" {
		switch op {
		case "context", "card", "reblogged_by", "favourited_by":
		default:
			return ErrInvalidParameter
		}

		endPoint += "/" + op
	}

	return mc.apiCall(endPoint, rest.Get, nil, data)
}

// updateStatusData updates the statuses
// The operation 'op' can be empty or "status" (to post a status), "delete"
// (for deleting a status), "reblog", "unreblog", "favourite", "unfavourite".
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
			return fmt.Errorf("too many (>4) media IDs")
		}
	case "delete":
		method = rest.Delete
		if opts.ID < 1 {
			return ErrInvalidID
		}
		endPoint += "/" + strconv.Itoa(opts.ID)
	case "reblog", "unreblog", "favourite", "unfavourite":
		if opts.ID < 1 {
			return ErrInvalidID
		}
		endPoint += "/" + strconv.Itoa(opts.ID) + "/" + op
	default:
		return ErrInvalidParameter
	}

	// Form items for a new toot
	if op == "status" {
		params["status"] = opts.Status
		if opts.InReplyToID > 0 {
			params["in_reply_to_id"] = strconv.Itoa(opts.InReplyToID)
		}
		for i, id := range opts.MediaIDs {
			if id < 1 {
				return ErrInvalidID
			}
			qID := fmt.Sprintf("media_ids[%d]", i+1)
			params[qID] = strconv.Itoa(id)
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

	return mc.apiCall(endPoint, method, params, data)
}

// GetStatus returns a status
// The returned status can be nil if there is an error or if the
// requested ID does not exist.
func (mc *Client) GetStatus(statusID int) (*Status, error) {
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
func (mc *Client) GetStatusContext(statusID int) (*Context, error) {
	var context Context
	if err := mc.queryStatusData(statusID, "context", &context); err != nil {
		return nil, err
	}
	return &context, nil
}

// GetStatusCard returns a status card
func (mc *Client) GetStatusCard(statusID int) (*Card, error) {
	var card Card
	if err := mc.queryStatusData(statusID, "card", &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// GetStatusRebloggedBy returns a list of the accounts who reblogged a status
func (mc *Client) GetStatusRebloggedBy(statusID int) ([]Account, error) {
	var accounts []Account
	err := mc.queryStatusData(statusID, "reblogged_by", &accounts)
	return accounts, err
}

// GetStatusFavouritedBy returns a list of the accounts who favourited a status
func (mc *Client) GetStatusFavouritedBy(statusID int) ([]Account, error) {
	var accounts []Account
	err := mc.queryStatusData(statusID, "favourited_by", &accounts)
	return accounts, err
}

// PostStatus posts a new "toot"
// All parameters but "text" can be empty.
// Visibility must be empty, or one of "direct", "private", "unlisted" and "public".
func (mc *Client) PostStatus(text string, inReplyTo int, mediaIDs []int, sensitive bool, spoilerText string, visibility string) (*Status, error) {
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
func (mc *Client) DeleteStatus(statusID int) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("delete", o, &status)
	return err
}

// ReblogStatus reblogs a status
func (mc *Client) ReblogStatus(statusID int) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("reblog", o, &status)
	return err
}

// UnreblogStatus unreblogs a status
func (mc *Client) UnreblogStatus(statusID int) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("unreblog", o, &status)
	return err
}

// FavouriteStatus favourites a status
func (mc *Client) FavouriteStatus(statusID int) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("favourite", o, &status)
	return err
}

// UnfavouriteStatus unfavourites a status
func (mc *Client) UnfavouriteStatus(statusID int) error {
	var status Status
	o := updateStatusOptions{ID: statusID}
	err := mc.updateStatusData("unfavourite", o, &status)
	return err
}
