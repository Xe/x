/*
Copyright 2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"github.com/sendgrid/rest"
)

// GetCustomEmojis returns a list with the server custom emojis
func (mc *Client) GetCustomEmojis(lopt *LimitParams) ([]Emoji, error) {
	var emojiList []Emoji
	if err := mc.apiCall("custom_emojis", rest.Get, nil, lopt, nil, &emojiList); err != nil {
		return nil, err
	}
	return emojiList, nil
}
