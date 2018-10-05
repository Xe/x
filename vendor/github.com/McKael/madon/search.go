/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"github.com/sendgrid/rest"
)

// Search search for contents (accounts or statuses) and returns a Results
func (mc *Client) Search(query string, resolve bool) (*Results, error) {
	if query == "" {
		return nil, ErrInvalidParameter
	}

	params := make(apiCallParams)
	params["q"] = query
	if resolve {
		params["resolve"] = "true"
	}

	var results Results
	if err := mc.apiCall("search", rest.Get, params, nil, nil, &results); err != nil {
		return nil, err
	}
	return &results, nil
}
