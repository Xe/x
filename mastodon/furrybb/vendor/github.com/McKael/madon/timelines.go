/*
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"fmt"
	"strings"

	"github.com/sendgrid/rest"
)

// GetTimelines returns a timeline (a list of statuses
// timeline can be "home", "public", or a hashtag (use ":hashtag" or "#hashtag")
// For the public timelines, you can set 'local' to true to get only the
// local instance.
func (mc *Client) GetTimelines(timeline string, local bool) ([]Status, error) {
	var endPoint string

	switch {
	case timeline == "home", timeline == "public":
		endPoint = "timelines/" + timeline
	case strings.HasPrefix(timeline, ":"), strings.HasPrefix(timeline, "#"):
		hashtag := timeline[1:]
		if hashtag == "" {
			return nil, fmt.Errorf("timelines API: empty hashtag")
		}
		endPoint = "timelines/tag/" + hashtag
	default:
		return nil, fmt.Errorf("GetTimelines: bad timelines argument")
	}

	params := make(apiCallParams)
	if timeline == "public" && local {
		params["local"] = "true"
	}

	var tl []Status
	if err := mc.apiCall(endPoint, rest.Get, params, &tl); err != nil {
		return nil, err
	}
	return tl, nil
}
