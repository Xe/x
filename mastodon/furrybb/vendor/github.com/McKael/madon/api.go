/*
Copyright 2017 Ollivier Robert
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sendgrid/rest"
)

// restAPI actually does the HTTP query
// It is a copy of rest.API with better handling of parameters with multiple values
func restAPI(request rest.Request) (*rest.Response, error) {
	c := &rest.Client{HTTPClient: http.DefaultClient}

	// Build the HTTP request object.
	if len(request.QueryParams) != 0 {
		// Add parameters to the URL
		request.BaseURL += "?"
		urlp := url.Values{}
		for key, value := range request.QueryParams {
			// It seems Mastodon doesn't like parameters with index
			// numbers, but it needs the brackets.
			// Let's check if the key matches '^.+\[.*\]$'
			klen := len(key)
			if klen == 0 {
				continue
			}
			i := strings.Index(key, "[")
			if key[klen-1] == ']' && i > 0 {
				// This is an array, let's remove the index number
				key = key[:i] + "[]"
			}
			urlp.Add(key, value)
		}
		urlpstr := urlp.Encode()
		request.BaseURL += urlpstr
	}

	req, err := http.NewRequest(string(request.Method), request.BaseURL, bytes.NewBuffer(request.Body))
	if err != nil {
		return nil, err
	}

	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}
	_, exists := req.Header["Content-Type"]
	if len(request.Body) > 0 && !exists {
		req.Header.Set("Content-Type", "application/json")
	}

	// Build the HTTP client and make the request.
	res, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	// Build Response object.
	response, err := rest.BuildResponse(res)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// prepareRequest inserts all pre-defined stuff
func (mc *Client) prepareRequest(target string, method rest.Method, params apiCallParams) (rest.Request, error) {
	var req rest.Request

	if mc == nil {
		return req, ErrUninitializedClient
	}

	endPoint := mc.APIBase + "/" + target

	// Request headers
	hdrs := make(map[string]string)
	hdrs["User-Agent"] = fmt.Sprintf("madon/%s", MadonVersion)
	if mc.UserToken != nil {
		hdrs["Authorization"] = fmt.Sprintf("Bearer %s", mc.UserToken.AccessToken)
	}

	req = rest.Request{
		BaseURL:     endPoint,
		Headers:     hdrs,
		Method:      method,
		QueryParams: params,
	}
	return req, nil
}

// apiCall makes a call to the Mastodon API server
func (mc *Client) apiCall(endPoint string, method rest.Method, params apiCallParams, data interface{}) error {
	if mc == nil {
		return fmt.Errorf("use of uninitialized madon client")
	}

	// Prepare query
	req, err := mc.prepareRequest(endPoint, method, params)
	if err != nil {
		return err
	}

	// Make API call
	r, err := restAPI(req)
	if err != nil {
		return fmt.Errorf("API query (%s) failed: %s", endPoint, err.Error())
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return fmt.Errorf("%s", errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	err = json.Unmarshal([]byte(r.Body), &data)
	if err != nil {
		return fmt.Errorf("cannot decode API response (%s): %s", method, err.Error())
	}
	return nil
}
