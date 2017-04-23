/*
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"github.com/sendgrid/rest"
)

// GetFavourites returns the list of the user's favourites
func (mc *Client) GetFavourites() ([]Status, error) {
	var faves []Status
	err := mc.apiCall("favourites", rest.Get, nil, &faves)
	if err != nil {
		return nil, err
	}
	return faves, nil
}
