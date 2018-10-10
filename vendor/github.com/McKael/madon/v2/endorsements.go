/*
Copyright 2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"github.com/sendgrid/rest"
)

// GetEndorsements returns the list of user's endorsements
func (mc *Client) GetEndorsements(lopt *LimitParams) ([]Account, error) {
	endPoint := "endorsements"
	method := rest.Get
	var accountList []Account
	if err := mc.apiCall("v1/"+endPoint, method, nil, lopt, nil, &accountList); err != nil {
		return nil, err
	}
	return accountList, nil
}

// PinAccount adds the account to the endorsement list
func (mc *Client) PinAccount(accountID int64) (*Relationship, error) {
	rel, err := mc.updateRelationship("pin", accountID, nil)
	if err != nil {
		return nil, err
	}
	if rel == nil {
		return nil, ErrEntityNotFound
	}
	return rel, nil
}

// UnpinAccount removes the account from the endorsement list
func (mc *Client) UnpinAccount(accountID int64) (*Relationship, error) {
	rel, err := mc.updateRelationship("unpin", accountID, nil)
	if err != nil {
		return nil, err
	}
	if rel == nil {
		return nil, ErrEntityNotFound
	}
	return rel, nil
}
