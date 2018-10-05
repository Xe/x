/*
Copyright 2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
)

// GetList returns a List entity
func (mc *Client) GetList(listID int64) (*List, error) {
	if listID <= 0 {
		return nil, errors.New("invalid list ID")
	}
	endPoint := "lists/" + strconv.FormatInt(listID, 10)
	var list List
	if err := mc.apiCall(endPoint, rest.Get, nil, nil, nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// GetLists returns a list of List entities
// If accountID is > 0, this will return the lists containing this account.
// If lopt.All is true, several requests will be made until the API server
// has nothing to return.
func (mc *Client) GetLists(accountID int64, lopt *LimitParams) ([]List, error) {
	endPoint := "lists"

	if accountID > 0 {
		endPoint = "accounts/" + strconv.FormatInt(accountID, 10) + "/lists"
	}

	var lists []List
	var links apiLinks
	if err := mc.apiCall(endPoint, rest.Get, nil, lopt, &links, &lists); err != nil {
		return nil, err
	}
	if lopt != nil { // Fetch more pages to reach our limit
		var listSlice []List
		for (lopt.All || lopt.Limit > len(lists)) && links.next != nil {
			newlopt := links.next
			links = apiLinks{}
			if err := mc.apiCall(endPoint, rest.Get, nil, newlopt, &links, &listSlice); err != nil {
				return nil, err
			}
			lists = append(lists, listSlice...)
			listSlice = listSlice[:0] // Clear struct
		}
	}
	return lists, nil
}

// CreateList creates a List
func (mc *Client) CreateList(title string) (*List, error) {
	params := apiCallParams{"title": title}
	method := rest.Post
	return mc.setSingleList(method, 0, params)
}

// UpdateList updates an existing List
func (mc *Client) UpdateList(listID int64, title string) (*List, error) {
	if listID <= 0 {
		return nil, errors.New("invalid list ID")
	}
	params := apiCallParams{"title": title}
	method := rest.Put
	return mc.setSingleList(method, listID, params)
}

// DeleteList deletes a list
func (mc *Client) DeleteList(listID int64) error {
	if listID <= 0 {
		return errors.New("invalid list ID")
	}
	method := rest.Delete
	_, err := mc.setSingleList(method, listID, nil)
	return err
}

// GetListAccounts returns the accounts belonging to a given list
func (mc *Client) GetListAccounts(listID int64, lopt *LimitParams) ([]Account, error) {
	endPoint := "lists/" + strconv.FormatInt(listID, 10) + "/accounts"
	return mc.getMultipleAccounts(endPoint, nil, lopt)
}

// AddListAccounts adds the accounts to a given list
func (mc *Client) AddListAccounts(listID int64, accountIDs []int64) error {
	endPoint := "lists/" + strconv.FormatInt(listID, 10) + "/accounts"
	method := rest.Post
	params := make(apiCallParams)
	for i, id := range accountIDs {
		if id < 1 {
			return ErrInvalidID
		}
		qID := fmt.Sprintf("account_ids[%d]", i+1)
		params[qID] = strconv.FormatInt(id, 10)
	}
	return mc.apiCall(endPoint, method, params, nil, nil, nil)
}

// RemoveListAccounts removes the accounts from the given list
func (mc *Client) RemoveListAccounts(listID int64, accountIDs []int64) error {
	endPoint := "lists/" + strconv.FormatInt(listID, 10) + "/accounts"
	method := rest.Delete
	params := make(apiCallParams)
	for i, id := range accountIDs {
		if id < 1 {
			return ErrInvalidID
		}
		qID := fmt.Sprintf("account_ids[%d]", i+1)
		params[qID] = strconv.FormatInt(id, 10)
	}
	return mc.apiCall(endPoint, method, params, nil, nil, nil)
}

func (mc *Client) setSingleList(method rest.Method, listID int64, params apiCallParams) (*List, error) {
	var endPoint string
	if listID > 0 {
		endPoint = "lists/" + strconv.FormatInt(listID, 10)
	} else {
		endPoint = "lists"
	}
	var list List
	if err := mc.apiCall(endPoint, method, params, nil, nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}
