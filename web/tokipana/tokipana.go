// Package tokipana wraps http://inamidst.com/services/tokipana.
package tokipana

import (
	"errors"
	"net/http"
	"net/url"
)

// The API URL.
const APIURL = `http://inamidst.com/services/tokipana`

// Errors
var (
	ErrInvalidRequest = errors.New("tokipana: invalid request")
)

// Validate checks if a response from the API is valid or not.
func Validate(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode%100 != 2 {
		return ErrInvalidRequest
	}

	return nil
}

// Translate returns a request to translate the given toki pona text into english.
func Translate(text string) *http.Request {
	u, _ := url.Parse(APIURL)
	q := u.Query()
	q.Set("text", text)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		panic(err)
	}

	return req
}
