/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"strings"

	"github.com/pkg/errors"
)

// GetTimelines returns a timeline (a list of statuses
// timeline can be "home", "public", a hashtag (use ":hashtag" or "#hashtag")
// or a list (use "!N", e.g. "!42" for list ID #42).
// For the public timelines, you can set 'local' to true to get only the
// local instance.
// Set 'onlyMedia' to true to only get statuses that have media attachments.
// If lopt.All is true, several requests will be made until the API server
// has nothing to return.
// If lopt.Limit is set (and not All), several queries can be made until the
// limit is reached.
func (mc *Client) GetTimelines(timeline string, local, onlyMedia bool, lopt *LimitParams) ([]Status, error) {
	var endPoint string

	switch {
	case timeline == "home", timeline == "public":
		endPoint = "timelines/" + timeline
	case strings.HasPrefix(timeline, ":"), strings.HasPrefix(timeline, "#"):
		hashtag := timeline[1:]
		if hashtag == "" {
			return nil, errors.New("timelines API: empty hashtag")
		}
		endPoint = "timelines/tag/" + hashtag
	case len(timeline) > 1 && strings.HasPrefix(timeline, "!"):
		// Check the timeline is a number
		for _, n := range timeline[1:] {
			if n < '0' || n > '9' {
				return nil, errors.New("timelines API: invalid list ID")
			}
		}
		endPoint = "timelines/list/" + timeline[1:]
	default:
		return nil, errors.New("GetTimelines: bad timelines argument")
	}

	params := make(apiCallParams)
	if timeline == "public" && local {
		params["local"] = "true"
	}
	if onlyMedia {
		params["only_media"] = "true"
	}

	return mc.getMultipleStatuses(endPoint, params, lopt)
}
