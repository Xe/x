package flymachines

import (
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
	result, err := doJSONBody[CreateAppArgs, CreateAppResponse](ctx, c, http.MethodPost, "/v1/apps", caa, http.StatusCreated)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't decode CreateApp response: %w", err)
	}

	return &result, nil
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
	result, err := doJSON[struct {
		Apps      []ListApp `json:"apps"`
		TotalApps int       `json:"total_apps"`
	}](ctx, c, http.MethodGet, "/v1/apps?org_slug="+orgSlug, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't decode GetApps response: %w", err)
	}

	return result.Apps, nil
}

// GetApp fetches information about one app in particular.
func (c *Client) GetApp(ctx context.Context, appName string) (*SingleApp, error) {
	result, err := doJSON[SingleApp](ctx, c, http.MethodGet, "/v1/apps/"+appName, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't decode GetApp response: %w", err)
	}

	return &result, nil
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
