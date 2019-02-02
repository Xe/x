// Package tokipana wraps http://inamidst.com/services/tokipana.
package tokipana

import (
	"net/http"
	"net/url"

	"github.com/Xe/x/web"
)

// The API URL.
const APIURL = `http://inamidst.com/services/tokipana`

// Validate checks if a response from the API is valid or not.
func Validate(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return web.NewError(http.StatusOK, resp)
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
