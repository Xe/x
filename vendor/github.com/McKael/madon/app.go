/*
Copyright 2017-2018 Mikael Berthe
Copyright 2017 Ollivier Robert

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
)

type registerApp struct {
	ID           int64  `json:"id,string"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// buildInstanceURL creates the URL from the instance name or cleans up the
// provided URL
func buildInstanceURL(instanceName string) (string, error) {
	if instanceName == "" {
		return "", errors.New("no instance provided")
	}

	instanceURL := instanceName
	if !strings.Contains(instanceURL, "/") {
		instanceURL = "https://" + instanceName
	}

	u, err := url.ParseRequestURI(instanceURL)
	if err != nil {
		return "", err
	}

	u.Path = ""
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// NewApp registers a new application with a given instance
func NewApp(name, website string, scopes []string, redirectURI, instanceName string) (mc *Client, err error) {
	instanceURL, err := buildInstanceURL(instanceName)
	if err != nil {
		return nil, err
	}

	mc = &Client{
		Name:        name,
		InstanceURL: instanceURL,
		APIBase:     instanceURL + currentAPIPath,
	}

	params := make(apiCallParams)
	params["client_name"] = name
	if website != "" {
		params["website"] = website
	}
	params["scopes"] = strings.Join(scopes, " ")
	if redirectURI != "" {
		params["redirect_uris"] = redirectURI
	} else {
		params["redirect_uris"] = NoRedirect
	}

	var app registerApp
	if err := mc.apiCall("apps", rest.Post, params, nil, nil, &app); err != nil {
		return nil, err
	}

	mc.ID = app.ClientID
	mc.Secret = app.ClientSecret

	return
}

// RestoreApp recreates an application client with existing secrets
func RestoreApp(name, instanceName, appID, appSecret string, userToken *UserToken) (mc *Client, err error) {
	instanceURL, err := buildInstanceURL(instanceName)
	if err != nil {
		return nil, err
	}

	return &Client{
		Name:        name,
		InstanceURL: instanceURL,
		APIBase:     instanceURL + currentAPIPath,
		ID:          appID,
		Secret:      appSecret,
		UserToken:   userToken,
	}, nil
}
