/*
Copyright 2017-2018 Mikael Berthe
Copyright 2017 Ollivier Robert

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
)

type apiLinks struct {
	next, prev *LimitParams
}

func parseLink(links []string) (*apiLinks, error) {
	if len(links) == 0 {
		return nil, nil
	}

	al := new(apiLinks)
	linkRegex := regexp.MustCompile(`<([^>]+)>; rel="([^"]+)`)
	for _, l := range links {
		m := linkRegex.FindAllStringSubmatch(l, -1)
		for _, submatch := range m {
			if len(submatch) != 3 {
				continue
			}
			// Parse URL
			u, err := url.Parse(submatch[1])
			if err != nil {
				return al, err
			}
			var lp *LimitParams
			since := u.Query().Get("since_id")
			max := u.Query().Get("max_id")
			lim := u.Query().Get("limit")
			if since == "" && max == "" {
				continue
			}
			lp = new(LimitParams)
			if since != "" {
				lp.SinceID, err = strconv.ParseInt(since, 10, 64)
				if err != nil {
					return al, err
				}
			}
			if max != "" {
				lp.MaxID, err = strconv.ParseInt(max, 10, 64)
				if err != nil {
					return al, err
				}
			}
			if lim != "" {
				lp.Limit, err = strconv.Atoi(lim)
				if err != nil {
					return al, err
				}
			}
			switch submatch[2] {
			case "prev":
				al.prev = lp
			case "next":
				al.next = lp
			}
		}
	}
	return al, nil
}

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
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errors.Errorf("bad server status code (%d): %s",
			res.StatusCode, http.StatusText(res.StatusCode))
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
// If links is not nil, the prev/next links from the API response headers
// will be set (if they exist) in the structure.
func (mc *Client) apiCall(endPoint string, method rest.Method, params apiCallParams, limitOptions *LimitParams, links *apiLinks, data interface{}) error {
	if mc == nil {
		return errors.New("use of uninitialized madon client")
	}

	if limitOptions != nil {
		if params == nil {
			params = make(apiCallParams)
		}
		if limitOptions.Limit > 0 {
			params["limit"] = strconv.Itoa(limitOptions.Limit)
		}
		if limitOptions.SinceID > 0 {
			params["since_id"] = strconv.FormatInt(limitOptions.SinceID, 10)
		}
		if limitOptions.MaxID > 0 {
			params["max_id"] = strconv.FormatInt(limitOptions.MaxID, 10)
		}
	}

	// Prepare query
	req, err := mc.prepareRequest(endPoint, method, params)
	if err != nil {
		return err
	}

	// Make API call
	r, err := restAPI(req)
	if err != nil {
		return errors.Wrapf(err, "API query (%s) failed", endPoint)
	}

	if links != nil {
		pLinks, err := parseLink(r.Headers["Link"])
		if err != nil {
			return errors.Wrapf(err, "cannot decode header links (%s)", method)
		}
		if pLinks != nil {
			*links = *pLinks
		}
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return errors.New(errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	err = json.Unmarshal([]byte(r.Body), &data)
	if err != nil {
		return errors.Wrapf(err, "cannot decode API response (%s)", method)
	}
	return nil
}
