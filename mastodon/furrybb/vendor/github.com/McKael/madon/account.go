/*
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/sendgrid/rest"
)

// getAccountsOptions contains option fields for POST and DELETE API calls
type getAccountsOptions struct {
	// The ID is used for most commands
	ID int

	// The following fields are used when searching for accounts
	Q     string
	Limit int
}

// getSingleAccount returns an account entity
// The operation 'op' can be "account", "verify_credentials", "follow",
// "unfollow", "block", "unblock", "mute", "unmute",
// "follow_requests/authorize" or // "follow_requests/reject".
// The id is optional and depends on the operation.
func (mc *Client) getSingleAccount(op string, id int) (*Account, error) {
	var endPoint string
	method := rest.Get
	strID := strconv.Itoa(id)

	switch op {
	case "account":
		endPoint = "accounts/" + strID
	case "verify_credentials":
		endPoint = "accounts/verify_credentials"
	case "follow", "unfollow", "block", "unblock", "mute", "unmute":
		endPoint = "accounts/" + strID + "/" + op
		method = rest.Post
	case "follow_requests/authorize", "follow_requests/reject":
		// The documentation is incorrect, the endpoint actually
		// is "follow_requests/:id/{authorize|reject}"
		endPoint = op[:16] + strID + "/" + op[16:]
		method = rest.Post
	default:
		return nil, ErrInvalidParameter
	}

	var account Account
	if err := mc.apiCall(endPoint, method, nil, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// getMultipleAccounts returns a list of account entities
// The operation 'op' can be "followers", "following", "search", "blocks",
// "mutes", "follow_requests".
// The id is optional and depends on the operation.
func (mc *Client) getMultipleAccounts(op string, opts *getAccountsOptions) ([]Account, error) {
	var endPoint string

	switch op {
	case "followers", "following":
		if opts == nil || opts.ID < 1 {
			return []Account{}, ErrInvalidID
		}
		endPoint = "accounts/" + strconv.Itoa(opts.ID) + "/" + op
	case "follow_requests", "blocks", "mutes":
		endPoint = op
	case "search":
		if opts == nil || opts.Q == "" {
			return []Account{}, ErrInvalidParameter
		}
		endPoint = "accounts/" + op
	default:
		return nil, ErrInvalidParameter
	}

	// Handle target-specific query parameters
	params := make(apiCallParams)
	if op == "search" {
		params["q"] = opts.Q
		if opts.Limit > 0 {
			params["limit"] = strconv.Itoa(opts.Limit)
		}
	}

	var accounts []Account
	if err := mc.apiCall(endPoint, rest.Get, params, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetAccount returns an account entity
// The returned value can be nil if there is an error or if the
// requested ID does not exist.
func (mc *Client) GetAccount(accountID int) (*Account, error) {
	account, err := mc.getSingleAccount("account", accountID)
	if err != nil {
		return nil, err
	}
	if account != nil && account.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return account, nil
}

// GetCurrentAccount returns the current user account
func (mc *Client) GetCurrentAccount() (*Account, error) {
	account, err := mc.getSingleAccount("verify_credentials", 0)
	if err != nil {
		return nil, err
	}
	if account != nil && account.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return account, nil
}

// GetAccountFollowers returns the list of accounts following a given account
func (mc *Client) GetAccountFollowers(accountID int) ([]Account, error) {
	o := &getAccountsOptions{ID: accountID}
	return mc.getMultipleAccounts("followers", o)
}

// GetAccountFollowing returns the list of accounts a given account is following
func (mc *Client) GetAccountFollowing(accountID int) ([]Account, error) {
	o := &getAccountsOptions{ID: accountID}
	return mc.getMultipleAccounts("following", o)
}

// FollowAccount follows an account
func (mc *Client) FollowAccount(accountID int) error {
	account, err := mc.getSingleAccount("follow", accountID)
	if err != nil {
		return err
	}
	if account != nil && account.ID != accountID {
		return ErrEntityNotFound
	}
	return nil
}

// UnfollowAccount unfollows an account
func (mc *Client) UnfollowAccount(accountID int) error {
	account, err := mc.getSingleAccount("unfollow", accountID)
	if err != nil {
		return err
	}
	if account != nil && account.ID != accountID {
		return ErrEntityNotFound
	}
	return nil
}

// FollowRemoteAccount follows a remote account
// The parameter 'uri' is a URI (e.mc. "username@domain").
func (mc *Client) FollowRemoteAccount(uri string) (*Account, error) {
	if uri == "" {
		return nil, ErrInvalidID
	}

	params := make(apiCallParams)
	params["uri"] = uri

	var account Account
	if err := mc.apiCall("follows", rest.Post, params, &account); err != nil {
		return nil, err
	}
	if account.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return &account, nil
}

// BlockAccount blocks an account
func (mc *Client) BlockAccount(accountID int) error {
	account, err := mc.getSingleAccount("block", accountID)
	if err != nil {
		return err
	}
	if account != nil && account.ID != accountID {
		return ErrEntityNotFound
	}
	return nil
}

// UnblockAccount unblocks an account
func (mc *Client) UnblockAccount(accountID int) error {
	account, err := mc.getSingleAccount("unblock", accountID)
	if err != nil {
		return err
	}
	if account != nil && account.ID != accountID {
		return ErrEntityNotFound
	}
	return nil
}

// MuteAccount mutes an account
func (mc *Client) MuteAccount(accountID int) error {
	account, err := mc.getSingleAccount("mute", accountID)
	if err != nil {
		return err
	}
	if account != nil && account.ID != accountID {
		return ErrEntityNotFound
	}
	return nil
}

// UnmuteAccount unmutes an account
func (mc *Client) UnmuteAccount(accountID int) error {
	account, err := mc.getSingleAccount("unmute", accountID)
	if err != nil {
		return err
	}
	if account != nil && account.ID != accountID {
		return ErrEntityNotFound
	}
	return nil
}

// SearchAccounts returns a list of accounts matching the query string
// The limit parameter is optional (can be 0).
func (mc *Client) SearchAccounts(query string, limit int) ([]Account, error) {
	o := &getAccountsOptions{Q: query, Limit: limit}
	return mc.getMultipleAccounts("search", o)
}

// GetBlockedAccounts returns the list of blocked accounts
func (mc *Client) GetBlockedAccounts() ([]Account, error) {
	return mc.getMultipleAccounts("blocks", nil)
}

// GetMutedAccounts returns the list of muted accounts
func (mc *Client) GetMutedAccounts() ([]Account, error) {
	return mc.getMultipleAccounts("mutes", nil)
}

// GetAccountFollowRequests returns the list of follow requests accounts
func (mc *Client) GetAccountFollowRequests() ([]Account, error) {
	return mc.getMultipleAccounts("follow_requests", nil)
}

// GetAccountRelationships returns a list of relationship entities for the given accounts
func (mc *Client) GetAccountRelationships(accountIDs []int) ([]Relationship, error) {
	if len(accountIDs) < 1 {
		return nil, ErrInvalidID
	}

	params := make(apiCallParams)
	for i, id := range accountIDs {
		if id < 1 {
			return nil, ErrInvalidID
		}
		qID := fmt.Sprintf("id[%d]", i+1)
		params[qID] = strconv.Itoa(id)
	}

	var rl []Relationship
	if err := mc.apiCall("accounts/relationships", rest.Get, params, &rl); err != nil {
		return nil, err
	}
	return rl, nil
}

// GetAccountStatuses returns a list of status entities for the given account
// If onlyMedia is true, returns only statuses that have media attachments.
// If excludeReplies is true, skip statuses that reply to other statuses.
func (mc *Client) GetAccountStatuses(accountID int, onlyMedia, excludeReplies bool) ([]Status, error) {
	if accountID < 1 {
		return nil, ErrInvalidID
	}

	endPoint := "accounts/" + strconv.Itoa(accountID) + "/" + "statuses"
	params := make(apiCallParams)
	if onlyMedia {
		params["only_media"] = "true"
	}
	if excludeReplies {
		params["exclude_replies"] = "true"
	}

	var sl []Status
	if err := mc.apiCall(endPoint, rest.Get, params, &sl); err != nil {
		return nil, err
	}
	return sl, nil
}

// FollowRequestAuthorize authorizes or rejects an account follow-request
func (mc *Client) FollowRequestAuthorize(accountID int, authorize bool) error {
	endPoint := "follow_requests/reject"
	if authorize {
		endPoint = "follow_requests/authorize"
	}
	_, err := mc.getSingleAccount(endPoint, accountID)
	return err
}

// UpdateAccount updates the connected user's account data
// The fields avatar & headerImage can contain base64-encoded images; if
// they do not (that is; if they don't contain ";base64,"), they are considered
// as file paths and their content will be encoded.
// All fields can be nil, in which case they are not updated.
// displayName and note can be set to "" to delete previous values;
// I'm not sure images can be deleted -- only replaced AFAICS.
func (mc *Client) UpdateAccount(displayName, note, avatar, headerImage *string) (*Account, error) {
	const endPoint = "accounts/update_credentials"
	params := make(apiCallParams)

	if displayName != nil {
		params["display_name"] = *displayName
	}
	if note != nil {
		params["note"] = *note
	}

	var err error
	avatar, err = fileToBase64(avatar, nil)
	if err != nil {
		return nil, err
	}
	headerImage, err = fileToBase64(headerImage, nil)
	if err != nil {
		return nil, err
	}

	var formBuf bytes.Buffer
	w := multipart.NewWriter(&formBuf)

	if avatar != nil {
		w.WriteField("avatar", *avatar)
	}
	if headerImage != nil {
		w.WriteField("header", *headerImage)
	}
	w.Close()

	// Prepare the request
	req, err := mc.prepareRequest(endPoint, rest.Patch, params)
	if err != nil {
		return nil, fmt.Errorf("prepareRequest failed: %s", err.Error())
	}
	req.Headers["Content-Type"] = w.FormDataContentType()
	req.Body = formBuf.Bytes()

	// Make API call
	r, err := restAPI(req)
	if err != nil {
		return nil, fmt.Errorf("account update failed: %s", err.Error())
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return nil, fmt.Errorf("%s", errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	var account Account
	if err := json.Unmarshal([]byte(r.Body), &account); err != nil {
		return nil, fmt.Errorf("cannot decode API response: %s", err.Error())
	}
	return &account, nil
}

// fileToBase64 is a helper function to convert a file's contents to
// base64-encoded data.  Is the data string already contains base64 data, it
// is not modified.
// If contentType is nil, it is detected.
func fileToBase64(data, contentType *string) (*string, error) {
	if data == nil {
		return nil, nil
	}

	if *data == "" {
		return data, nil
	}

	if strings.Contains(*data, ";base64,") {
		return data, nil
	}

	// We need to convert the file and file name to base64

	file, err := os.Open(*data)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fStat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, fStat.Size())
	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}

	var cType string
	if contentType == nil || *contentType == "" {
		cType = http.DetectContentType(buffer[:512])
	} else {
		cType = *contentType
	}
	contentData := base64.StdEncoding.EncodeToString(buffer)
	newData := "data:" + cType + ";base64," + contentData
	return &newData, nil
}
