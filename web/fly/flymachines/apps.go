package flymachines

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"within.website/x/web"
)

// CreateAppArgs are the arguments to the CreateApp call.
type CreateAppArgs struct {
	AppName string `json:"app_name"`
	Network string `json:"network"`
	OrgSlug string `json:"org_slug"`
}

// CreateAppResponse is the response from the CreateApp call.
type CreateAppResponse struct {
	ID        string    `json:"id"`
	CreatedAt MilliTime `json:"created_at"`
}

// MilliTime is a time.Time that can be marshalled and unmarshalled from milliseconds since the Unix epoch.
type MilliTime struct {
	time.Time
}

func (mt *MilliTime) UnmarshalJSON(b []byte) error {
	var millis int64
	err := json.Unmarshal(b, &millis)
	if err != nil {
		return err
	}

	mt.Time = time.UnixMilli(millis)
	return nil
}

func (mt MilliTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(mt.Time.UnixMilli())
}

// CreateApp creates a single application in the given organization and on the given network.
func (c *Client) CreateApp(ctx context.Context, caa CreateAppArgs) (*CreateAppResponse, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(caa); err != nil {
		return nil, fmt.Errorf("flymachines: can't encode CreateApp request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/v1/apps", &buf)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't create CreateApp request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't perform CreateApp request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, web.NewError(http.StatusCreated, resp)
	}

	var car CreateAppResponse

	if err := json.NewDecoder(resp.Body).Decode(&car); err != nil {
		return nil, fmt.Errorf("flymachines: can't decode response body for CreateApp: %w", err)
	}

	return &car, nil
}

// App is a Fly app. Apps are collections of resources such as machines, volumes, and IP addresses.
type App struct {
	ID   string `json:"id"`   // The unique ID of the app
	Name string `json:"name"` // The name of the app (also unique but human readable)
}

// ListApp is a Fly app with extra information that is only shown when you're listing apps with GetApps.
type ListApp struct {
	App
	MachineCount int    `json:"machine_count"` // The number of machines associated with this app
	Network      string `json:"network"`       // The network this app is on
}

// SingleApp is a Fly app with extra information that is only shown when you're getting a single app with GetApp.
type SingleApp struct {
	App
	Organization Org    `json:"organization"` // The organization this app belongs to
	Status       string `json:"status"`       // The current status of the app
}

// Org is a Fly organization. An organization is a collection of apps and users that are allowed to manage
// that collection.
type Org struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetApps gets all of the applications in an organization.
func (c *Client) GetApps(ctx context.Context, orgSlug string) ([]ListApp, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/v1/apps", nil)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't create GetApps request: %w", err)
	}

	q := req.URL.Query()
	q.Set("org_slug", orgSlug)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't perform GetApps request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result struct {
		Apps      []ListApp `json:"apps"`
		TotalApps int       `json:"total_apps"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("flymachines: can't decode response body for GetApps: %w", err)
	}

	return result.Apps, nil
}

// GetApp fetches information about one app in particular.
func (c *Client) GetApp(ctx context.Context, appName string) (*SingleApp, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/v1/apps/"+appName, nil)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't create GetApp request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't perform GetApp request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var app SingleApp
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("flymachines: can't decode response body for GetApp: %w", err)
	}

	return &app, nil
}

func (c *Client) DeleteApp(ctx context.Context, appName string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.apiURL+"/v1/apps/"+appName, nil)
	if err != nil {
		return fmt.Errorf("flymachines: can't create DeleteApp request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("flymachines: can't perform DeleteApp request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return web.NewError(http.StatusAccepted, resp)
	}

	return nil
}
