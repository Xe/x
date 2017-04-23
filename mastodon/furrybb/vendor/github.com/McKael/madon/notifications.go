/*
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"strconv"

	"github.com/sendgrid/rest"
)

// GetNotifications returns the list of the user's notifications
func (mc *Client) GetNotifications() ([]Notification, error) {
	var notifications []Notification
	if err := mc.apiCall("notifications", rest.Get, nil, &notifications); err != nil {
		return nil, err
	}
	return notifications, nil
}

// GetNotification returns a notification
// The returned notification can be nil if there is an error or if the
// requested notification does not exist.
func (mc *Client) GetNotification(notificationID int) (*Notification, error) {
	if notificationID < 1 {
		return nil, ErrInvalidID
	}

	var endPoint = "notifications/" + strconv.Itoa(notificationID)
	var notification Notification
	if err := mc.apiCall(endPoint, rest.Get, nil, &notification); err != nil {
		return nil, err
	}
	if notification.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return &notification, nil
}

// ClearNotifications deletes all notifications from the Mastodon server for
// the authenticated user
func (mc *Client) ClearNotifications() error {
	return mc.apiCall("notifications/clear", rest.Post, nil, &Notification{})
}
