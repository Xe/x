/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"github.com/sendgrid/rest"
)

// GetCurrentInstance returns current instance information
func (mc *Client) GetCurrentInstance() (*Instance, error) {
	var i Instance
	if err := mc.apiCall("v1/instance", rest.Get, nil, nil, nil, &i); err != nil {
		return nil, err
	}
	return &i, nil
}

// GetInstancePeers returns current instance peers
// The peers are defined as the domains of users the instance has previously
// resolved.
func (mc *Client) GetInstancePeers() ([]InstancePeer, error) {
	var peers []InstancePeer
	if err := mc.apiCall("v1/instance/peers", rest.Get, nil, nil, nil, &peers); err != nil {
		return nil, err
	}
	return peers, nil
}

// GetInstanceActivity returns current instance activity
// The activity contains the counts of active users, locally posted statuses,
// and new registrations in weekly buckets.
func (mc *Client) GetInstanceActivity() ([]WeekActivity, error) {
	var activity []WeekActivity
	if err := mc.apiCall("v1/instance/activity", rest.Get, nil, nil, nil, &activity); err != nil {
		return nil, err
	}
	return activity, nil
}
