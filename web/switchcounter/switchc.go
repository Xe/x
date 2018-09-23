// Package switchcounter is a simple interface to the https://www.switchcounter.science/ API.
package switchcounter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type arg struct {
	Command    string `json:"command"` // always "switch"
	MemberName string `json:"member_name,omitempty"`
}

// API is the switchcounter API as an abstract interface.
type API interface {
	// Status returns the front of the system for this API client.
	Status(ctx context.Context) (Status, error)

	// Switch changes who is in front.
	Switch(ctx context.Context, front string) (Status, error)
}

// Status is the API response.
type Status struct {
	Front     string    `json:"member_name"`
	StartedAt time.Time `json:"started_at"`
}

type httpClient struct {
	hc  *http.Client
	url string // webhook url
}

func (hc httpClient) makeRequestWith(ctx context.Context, body interface{}) (*Status, error) {
	env := struct {
		Webhook interface{} `json:"webhook"`
	}{
		Webhook: body,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", hc.url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Add("Content-Type", "application/json")

	resp, err := hc.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		log.Printf("body: %s", string(data))

		return nil, fmt.Errorf("http response code %d", resp.StatusCode)
	}

	var result Status
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (hc httpClient) Status(ctx context.Context) (Status, error) {
	result, err := hc.makeRequestWith(ctx, arg{Command: "switch"})
	if err != nil {
		return Status{}, err
	}
	return *result, nil
}

func (hc httpClient) Switch(ctx context.Context, front string) (Status, error) {
	result, err := hc.makeRequestWith(ctx, arg{Command: "switch", MemberName: front})
	if err != nil {
		return Status{}, err
	}
	return *result, nil
}

// NewHTTPClient creates a new instance of API over HTTP.
func NewHTTPClient(hc *http.Client, webhookURL string) API {
	return httpClient{
		hc:  hc,
		url: webhookURL,
	}
}
