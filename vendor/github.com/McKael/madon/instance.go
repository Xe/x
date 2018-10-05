/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sendgrid/rest"
)

// GetCurrentInstance returns current instance information
func (mc *Client) GetCurrentInstance() (*Instance, error) {
	var i Instance
	if err := mc.apiCall("instance", rest.Get, nil, nil, nil, &i); err != nil {
		return nil, err
	}
	return &i, nil
}

// GetInstancePeers returns current instance peers
// The peers are defined as the domains of users the instance has previously
// resolved.
func (mc *Client) GetInstancePeers() ([]InstancePeer, error) {
	var peers []InstancePeer
	if err := mc.apiCall("instance/peers", rest.Get, nil, nil, nil, &peers); err != nil {
		return nil, err
	}
	return peers, nil
}

// GetInstanceActivity returns current instance activity
func (mc *Client) GetInstanceActivity() ([]WeekActivity, error) {
	var activity []WeekActivity
	if err := mc.apiCall("instance/activity", rest.Get, nil, nil, nil, &activity); err != nil {
		return nil, err
	}
	return activity, nil
}

/* Activity time handling */

// UnmarshalJSON handles deserialization for custom ActivityTime type
func (act *ActivityTime) UnmarshalJSON(b []byte) error {
	s, err := strconv.ParseInt(strings.Trim(string(b), "\""), 10, 64)
	if err != nil {
		return err
	}
	if s == 0 {
		act.Time = time.Time{}
		return nil
	}
	act.Time = time.Unix(s, 0)
	return nil
}

// MarshalJSON handles serialization for custom ActivityTime type
func (act *ActivityTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%d\"", act.Unix())), nil
}
