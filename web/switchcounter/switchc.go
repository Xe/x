// Package switchcounter is a simple interface to the https://www.switchcounter.science/ API.
package switchcounter

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type arg struct {
	Command    string `json:"command"` // always "switch"
	MemberName string `json:"member_name,omitempty"`
}

// Status is the API response.
type Status struct {
	Front     string    `json:"member_name"`
	StartedAt time.Time `json:"started_at"`
}

type API struct {
	url string // webhook url
}

func (a API) makeRequestWith(body interface{}) (*http.Request, error) {
	env := struct {
		Webhook interface{} `json:"webhook"`
	}{
		Webhook: body,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, a.url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	return req, nil
}

func (a API) Status() *http.Request {
	result, err := a.makeRequestWith(arg{Command: "switch"})
	if err != nil {
		panic(err)
	}
	return result
}

func (a API) Switch(front string) *http.Request {
	result, err := a.makeRequestWith(arg{Command: "switch", MemberName: front})
	if err != nil {
		panic(err)
	}
	return result
}

// NewHTTPClient creates a new instance of API over HTTP.
func NewHTTPClient(a *http.Client, webhookURL string) API {
	return API{
		url: webhookURL,
	}
}
