/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"strings"

	"github.com/sendgrid/rest"
)

// Search search for contents (accounts or statuses) and returns a Results
func (mc *Client) searchV1(params apiCallParams) (*Results, error) {
	// We use a custom structure with shadowed Hashtags field,
	// since the v1 version only returns strings.
	var resultsV1 struct {
		Results
		Hashtags []string `json:"hashtags"`
	}
	if err := mc.apiCall("v1/"+"search", rest.Get, params, nil, nil, &resultsV1); err != nil {
		return nil, err
	}

	var results Results
	results.Accounts = resultsV1.Accounts
	results.Statuses = resultsV1.Statuses
	for _, t := range resultsV1.Hashtags {
		results.Hashtags = append(results.Hashtags, Tag{Name: t})
	}

	return &results, nil
}

func (mc *Client) searchV2(params apiCallParams) (*Results, error) {
	var results Results
	if err := mc.apiCall("v2/"+"search", rest.Get, params, nil, nil, &results); err != nil {
		return nil, err
	}

	return &results, nil
}

// Search search for contents (accounts or statuses) and returns a Results
func (mc *Client) Search(query string, resolve bool) (*Results, error) {
	if query == "" {
		return nil, ErrInvalidParameter
	}

	// The parameters are the same in both v1 & v2 API versions
	params := make(apiCallParams)
	params["q"] = query
	if resolve {
		params["resolve"] = "true"
	}

	r, err := mc.searchV2(params)

	// This is not a very beautiful way to check the error cause, I admit.
	if err != nil && strings.Contains(err.Error(), "bad server status code (404)") {
		// Fall back to v1 API endpoint
		r, err = mc.searchV1(params)
	}

	return r, err
}
