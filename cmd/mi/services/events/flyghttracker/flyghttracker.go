package flyghttracker

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"within.website/x/web"
)

var (
	flyghttrackerURL = flag.String("flyghttracker-url", "https://flyght-tracker.fly.dev/api/upcoming_events", "Flyghttracker URL")
)

// Date represents a date in the format "YYYY-MM-DD"
type Date struct {
	time.Time
}

// UnmarshalJSON parses a JSON string in the format "YYYY-MM-DD" to a Date
func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

// MarshalJSON returns a JSON string in the format "YYYY-MM-DD"
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time.Format("2006-01-02"))
}

// Event represents an event that members of DevRel will be attending.
type Event struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name,omitempty"`
	URL       string   `json:"url,omitempty"`
	StartDate Date     `json:"start_date,omitempty"`
	EndDate   Date     `json:"end_date,omitempty"`
	Location  string   `json:"location,omitempty"`
	People    []string `json:"people,omitempty"`
}

type Error struct {
	Code   int
	Detail string `json:"detail"`
}

func (e Error) LogValues() slog.Value {
	return slog.GroupValue(
		slog.Int("code", e.Code),
		slog.String("detail", e.Detail),
	)
}

func (e Error) Error() string {
	return fmt.Sprintf("flyghttracker: %d %s", e.Code, e.Detail)
}

type Client struct {
	URL string
}

// New creates a new Flyght Tracker client.
func New(url string) *Client {
	return &Client{
		URL: url,
	}
}

// Create creates a new Flyght Tracker event.
func (c *Client) Create(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/api/events", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var e Error
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			return fmt.Errorf("failed to decode error: %w", err)
		}

		e.Code = resp.StatusCode

		return e
	}

	return nil
}

// Fetch new events from the Flyght Tracker URL.
//
// It returns a list of events that end in the future and that have "Xe" as one of the attendees.
func (c *Client) Fetch() ([]Event, error) {
	resp, err := http.Get(c.URL + "/api/upcoming_events")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch flyghttracker events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var events []Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode flyghttracker events: %w", err)
	}

	var result []Event

	for _, event := range events {
		if event.EndDate.Before(time.Now()) {
			continue
		}

		found := false
		for _, person := range event.People {
			if person == "Xe" {
				found = true
				break
			}
		}

		if found {
			result = append(result, event)
		}
	}

	return result, nil
}
