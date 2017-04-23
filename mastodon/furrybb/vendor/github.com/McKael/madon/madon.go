/*
Copyright 2017 Ollivier Robert
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"errors"
)

// apiCallParams is a map with the parameters for an API call
type apiCallParams map[string]string

const (
	// MadonVersion contains the version of the Madon library
	MadonVersion = "0.2"

	// API version implemented in this library
	apiVersion     = "v1"
	currentAPIPath = "/api/" + apiVersion

	// NoRedirect is the URI for no redirection in the App registration
	NoRedirect = "urn:ietf:wg:oauth:2.0:oob"
)

// Error codes
var (
	ErrUninitializedClient = errors.New("use of uninitialized madon client")
	ErrAlreadyRegistered   = errors.New("app already registered")
	ErrEntityNotFound      = errors.New("entity not found")
	ErrInvalidParameter    = errors.New("incorrect parameter")
	ErrInvalidID           = errors.New("incorrect entity ID")
)
