/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"fmt"
	"strconv"

	"github.com/sendgrid/rest"
)

// GetNotifications returns the list of the user's notifications
// excludeTypes is an array of notifications to exclude ("follow", "favourite",
// "reblog", "mention").  It can be nil.
// If lopt.All is true, several requests will be made until the API server
// has nothing to return.
// If lopt.Limit is set (and not All), several queries can be made until the
// limit is reached.
func (mc *Client) GetNotifications(excludeTypes []string, lopt *LimitParams) ([]Notification, error) {
	var notifications []Notification
	var links apiLinks
	var params apiCallParams

	if len(excludeTypes) > 0 {
		params = make(apiCallParams)
		for i, eType := range excludeTypes {
			qID := fmt.Sprintf("exclude_types[%d]", i+1)
			params[qID] = eType
		}
	}

	if err := mc.apiCall("notifications", rest.Get, params, lopt, &links, &notifications); err != nil {
		return nil, err
	}
	if lopt != nil { // Fetch more pages to reach our limit
		var notifSlice []Notification
		for (lopt.All || lopt.Limit > len(notifications)) && links.next != nil {
			newlopt := links.next
			links = apiLinks{}
			if err := mc.apiCall("notifications", rest.Get, nil, newlopt, &links, &notifSlice); err != nil {
				return nil, err
			}
			notifications = append(notifications, notifSlice...)
			notifSlice = notifSlice[:0] // Clear struct
		}
	}
	return notifications, nil
}

// GetNotification returns a notification
// The returned notification can be nil if there is an error or if the
// requested notification does not exist.
func (mc *Client) GetNotification(notificationID int64) (*Notification, error) {
	if notificationID < 1 {
		return nil, ErrInvalidID
	}

	var endPoint = "notifications/" + strconv.FormatInt(notificationID, 10)
	var notification Notification
	if err := mc.apiCall(endPoint, rest.Get, nil, nil, nil, &notification); err != nil {
		return nil, err
	}
	if notification.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return &notification, nil
}

// DismissNotification deletes a notification
func (mc *Client) DismissNotification(notificationID int64) error {
	if notificationID < 1 {
		return ErrInvalidID
	}

	endPoint := "notifications/dismiss"
	params := apiCallParams{"id": strconv.FormatInt(notificationID, 10)}
	err := mc.apiCall(endPoint, rest.Post, params, nil, nil, &Notification{})
	return err
}

// ClearNotifications deletes all notifications from the Mastodon server for
// the authenticated user
func (mc *Client) ClearNotifications() error {
	err := mc.apiCall("notifications/clear", rest.Post, nil, nil, nil, &Notification{})
	return err
}
