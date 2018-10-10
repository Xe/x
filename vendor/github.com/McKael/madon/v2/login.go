/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
)

const oAuthRelPath = "/oauth/"

// UserToken represents a user token as returned by the Mastodon API
type UserToken struct {
	AccessToken string `json:"access_token"`
	CreatedAt   int64  `json:"created_at"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

// LoginBasic does basic user authentication
func (mc *Client) LoginBasic(username, password string, scopes []string) error {
	if mc == nil {
		return ErrUninitializedClient
	}

	if username == "" {
		return errors.New("missing username")
	}
	if password == "" {
		return errors.New("missing password")
	}

	hdrs := make(map[string]string)
	opts := make(map[string]string)

	hdrs["User-Agent"] = "madon/" + MadonVersion

	opts["grant_type"] = "password"
	opts["client_id"] = mc.ID
	opts["client_secret"] = mc.Secret
	opts["username"] = username
	opts["password"] = password
	if len(scopes) > 0 {
		opts["scope"] = strings.Join(scopes, " ")
	}

	req := rest.Request{
		BaseURL:     mc.InstanceURL + oAuthRelPath + "token",
		Headers:     hdrs,
		QueryParams: opts,
		Method:      rest.Post,
	}

	r, err := restAPI(req)
	if err != nil {
		return err
	}

	var resp UserToken

	err = json.Unmarshal([]byte(r.Body), &resp)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal server response")
	}

	mc.UserToken = &resp
	return nil
}

// SetUserToken sets an existing user credentials
// No verification of the arguments is made.
func (mc *Client) SetUserToken(token, username, password string, scopes []string) error {
	if mc == nil {
		return ErrUninitializedClient
	}

	mc.UserToken = &UserToken{
		AccessToken: token,
		Scope:       strings.Join(scopes, " "),
		TokenType:   "bearer",
	}
	return nil
}

// LoginOAuth2 handles OAuth2 authentication
// If code is empty, the URL to the server consent page will be returned;
// if not, the user token is set.
func (mc *Client) LoginOAuth2(code string, scopes []string) (string, error) {
	if mc == nil {
		return "", ErrUninitializedClient
	}

	conf := &oauth2.Config{
		ClientID:     mc.ID,
		ClientSecret: mc.Secret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  mc.InstanceURL + oAuthRelPath + "authorize",
			TokenURL: mc.InstanceURL + oAuthRelPath + "token",
		},
		RedirectURL: NoRedirect,
	}

	if code == "" {
		// URL to consent page to ask for permission
		// for the scopes specified above.
		return conf.AuthCodeURL("state", oauth2.AccessTypeOffline), nil
	}

	// Return token
	t, err := conf.Exchange(context.TODO(), code)
	if err != nil {
		return "", errors.Wrap(err, "cannot convert code into a token")
	}
	if t == nil || t.AccessToken == "" {
		return "", errors.New("empty token")
	}
	return "", mc.SetUserToken(t.AccessToken, "", "", scopes)
}
