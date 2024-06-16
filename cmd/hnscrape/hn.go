package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"within.website/x/web"
)

// All of these types match descriptions in the Hacker News API documentation.
//
// https://github.com/HackerNews/API

// UnixTime is a type that represents a Unix timestamp.
type UnixTime time.Time

func (u UnixTime) String() string {
	return time.Time(u).Format(time.RFC3339)
}

// UnmarshalJSON converts the unix timestamp into a time.Time.
func (u *UnixTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")

	unix, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	*u = UnixTime(time.Unix(unix, 0))

	return nil
}

// MarshalJSON converts the time.Time into a unix timestamp.
func (u UnixTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(u).Unix(), 10)), nil
}

// HNUser is a type that represents a Hacker News user.
type HNUser struct {
	ID        string   `json:"id"`
	Created   UnixTime `json:"created"`
	Karma     int      `json:"karma"`
	About     string   `json:"about"` // HTML
	Submitted []int    `json:"submitted"`
}

// HNItem is a type that represents a Hacker News item. This is a superset of all other types of items.
type HNItem struct {
	ID          int      `json:"id"`
	Deleted     *bool    `json:"deleted,omitempty"`
	Type        string   `json:"type"`
	By          string   `json:"by"`
	Time        UnixTime `json:"time"`
	Text        string   `json:"text,omitempty"` // HTML
	Dead        *bool    `json:"dead,omitempty"`
	Parent      *int     `json:"parent,omitempty"`
	Poll        *int     `json:"poll,omitempty"`
	Kids        []int    `json:"kids,omitempty"`
	URL         string   `json:"url,omitempty"`
	Score       int      `json:"score,omitempty"`
	Title       string   `json:"title,omitempty"` // HTML
	Parts       []int    `json:"parts,omitempty"`
	Descendants *int     `json:"descendants,omitempty"`
}

type HNClient struct {
	t           *time.Ticker // for rate limiting requests based on --scrape-delay
	cli         *http.Client
	cacheFolder *string
}

func NewHNClient(delay time.Duration) *HNClient {
	return &HNClient{
		t:   time.NewTicker(delay),
		cli: &http.Client{},
	}
}

func (h *HNClient) Close() {
	h.t.Stop()
}

func (h *HNClient) WithClient(c *http.Client) *HNClient {
	h.cli = c
	return h
}

func (h *HNClient) WithCacheFolder(f string) *HNClient {
	h.cacheFolder = &f
	return h
}

func (h *HNClient) GetItem(ctx context.Context, id int) (*HNItem, error) {
	if h.cacheFolder != nil {
		item, err := h.getItemFromCache(id)
		if err == nil {
			return item, nil
		}
	}

	item, err := h.getItem(ctx, id)
	if err != nil {
		return nil, err
	}

	if h.cacheFolder != nil {
		err = h.saveItemToCache(item)
		if err != nil {
			return nil, err
		}
	}

	return item, nil
}

func (h *HNClient) saveItemToCache(item *HNItem) error {
	if h.cacheFolder == nil {
		return nil
	}

	folder := *h.cacheFolder

	fname := filepath.Join(folder, "items", strconv.Itoa(item.ID)+".json")
	f, err := os.Create(fname)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer f.Close()

	if err = json.NewEncoder(f).Encode(item); err != nil {
		return fmt.Errorf("failed to write item to cache: %w", err)
	}

	return nil
}

func (h *HNClient) getItemFromCache(id int) (*HNItem, error) {
	if h.cacheFolder == nil {
		return nil, nil
	}

	folder := *h.cacheFolder

	fname := filepath.Join(folder, "items", strconv.Itoa(id)+".json")
	f, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache file: %w", err)
	}
	defer f.Close()

	var item HNItem
	if err = json.NewDecoder(f).Decode(&item); err != nil {
		return nil, fmt.Errorf("failed to read item from cache: %w", err)
	}

	return &item, nil
}

func (h *HNClient) GetUser(ctx context.Context, id string) (*HNUser, error) {
	<-h.t.C

	req, err := http.NewRequestWithContext(ctx, "GET", "https://hacker-news.firebaseio.com/v0/user/"+id+".json", nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var user HNUser
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (h *HNClient) getItem(ctx context.Context, id int) (*HNItem, error) {
	<-h.t.C

	req, err := http.NewRequestWithContext(ctx, "GET", "https://hacker-news.firebaseio.com/v0/item/"+strconv.Itoa(id)+".json", nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var item HNItem
	err = json.NewDecoder(resp.Body).Decode(&item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (h *HNClient) GetUltimateParent(ctx context.Context, id int) (*HNItem, error) {
	item, err := h.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}

	if item.Parent == nil {
		return item, nil
	}

	return h.GetUltimateParent(ctx, *item.Parent)
}
