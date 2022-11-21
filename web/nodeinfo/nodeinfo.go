// Package nodeinfo contains types and a simple client for reading the standard NodeInfo protocol described by Diaspora[1].
//
// This package supports version 2.0[2] because that is the one most commonly used.
//
// [1]: http://nodeinfo.diaspora.software/
// [2]: http://nodeinfo.diaspora.software/docson/index.html#/ns/schema/2.0#$$expand
package nodeinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"within.website/x/web"
	"within.website/x/web/useragent"
)

const schema2point0 = "http://nodeinfo.diaspora.software/ns/schema/2.0"
const twoMebibytes = 1024 * 1024 * 2

// Node is the node information you are looking for.
type Node struct {
	Version          string   `json:"version"`
	Software         Software `json:"software"`
	Protocols        []string `json:"protocols"`
	Services         Services `json:"services"`
	OpenRegistration bool     `json:"openRegistration"`
	Usage            Usage    `json:"usage"`
	Metadata         any      `json:"metadata"`
}

// Software contains metadata about the server software in use.
type Software struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Services lists the third party services that this server can connect to with their application API.
type Services struct {
	Inbound  []string `json:"inbound"`
	Outbound []string `json:"outbound"`
}

// Usage statistics for this server.
type Usage struct {
	Users      Users `json:"users"`
	LocalPosts int64 `json:"localPosts"`
}

// Users contains statistics about the users of this server.
type Users struct {
	Total          int64 `json:"total"`
	ActiveHalfYear int64 `json:"activeHalfYear"`
	ActiveMonth    int64 `json:"activeMonth"`
}

type wellKnownLinks struct {
	Links []wellKnownLink `json:"links"`
}

type wellKnownLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// FetchWithClient uses the provided HTTP client to fetch node information.
func FetchWithClient(ctx context.Context, cli *http.Client, nodeURL string) (*Node, error) {
	u, err := url.Parse(nodeURL)
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't parse nodeURL: %w", err)
	}

	u.Path = "/.well-known/nodeinfo"
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't create HTTP request: %w", err)
	}

	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't read HTTP response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, twoMebibytes))
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't read from HTTP response body: %w", err)
	}

	var niw wellKnownLinks
	err = json.Unmarshal(data, &niw)
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't unmarshal discovery metadata: %w", err)
	}

	var targetURL string

	for _, link := range niw.Links {
		if link.Rel == schema2point0 {
			targetURL = link.Href
		}
	}

	if targetURL == "" {
		return nil, fmt.Errorf("nodeinfo: can't find schema 2.0 nodeinfo for %s", nodeURL)
	}

	req, err = http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't create HTTP request: %w", err)
	}

	resp, err = cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't read HTTP response: %w", err)
	}

	data, err = io.ReadAll(io.LimitReader(resp.Body, twoMebibytes))
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't read from HTTP response body: %w", err)
	}

	var result Node
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("nodeinfo: can't unmarshal nodeinfo: %w", err)
	}

	return &result, nil
}

// Fetch uses the standard library HTTP client to fetch node information.
func Fetch(ctx context.Context, nodeURL string) (*Node, error) {
	cli := &http.Client{
		Transport: useragent.Transport("github.com/Xe/x/web/nodeinfo", "https://within.website/.x.botinfo", http.DefaultTransport),
	}
	return FetchWithClient(ctx, cli, nodeURL)
}
