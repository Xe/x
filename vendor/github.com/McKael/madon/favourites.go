/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

// GetFavourites returns the list of the user's favourites
// If lopt.All is true, several requests will be made until the API server
// has nothing to return.
// If lopt.Limit is set (and not All), several queries can be made until the
// limit is reached.
func (mc *Client) GetFavourites(lopt *LimitParams) ([]Status, error) {
	return mc.getMultipleStatuses("favourites", nil, lopt)
}
