// Package ghstat is a set of wrappers for the GitHub Status API.
package ghstat

import "net/http"

// Status is the human-readable status for GitHub's platform.
type Status struct {
	Status      string `json:"status"`
	LastUpdated string `json:"last_updated"`
}

// GitHub status API constants.
const (
	GHStatusAPIRoot = `https://status.github.com/api/`
	StatusPath      = `status.json`
	MessagePath     = `last-message.json`
)

// LastStatus returns a request to the most recent status for GitHub's platform.
func LastStatus() *http.Request {
	req, err := http.NewRequest(http.MethodGet, GHStatusAPIRoot+StatusPath, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "application/json")

	return req
}

// Message is an individiual human-readable message associated with a status update.
type Message struct {
	Status    string `json:"status"`
	Body      string `json:"body"`
	CreatedOn string `json:"created_on"`
}

// LastMessage returns a request to the most recent message for GitHub's platform.
func LastMessage() *http.Request {
	req, err := http.NewRequest(http.MethodGet, GHStatusAPIRoot+StatusPath, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "application/json")

	return req
}
