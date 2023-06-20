// Package marginalia implements the Marginalia search API.
//
// You need an API key to use this. See the Marginalia API docs for more information: https://www.marginalia.nu/marginalia-search/api/
package marginalia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"within.website/x/web"
)

type Request struct {
	Query string
	Count *int
	Index *int
}

type Response struct {
	License string   `json:"license"`
	Query   string   `json:"query"`
	Results []Result `json:"results"`
}

type Result struct {
	URL         string     `json:"url"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Quality     float64    `json:"quality"`
	Details     [][]Detail `json:"details"`
}

type Detail struct {
	Keyword          string   `json:"keyword"`
	Count            int      `json:"count"`
	FlagsUnstableAPI []string `json:"flagsUnstableAPI"`
}

type Client struct {
	apiKey  string
	httpCli *http.Client
}

func New(apiKey string, httpCli *http.Client) *Client {
	if httpCli == nil {
		httpCli = &http.Client{}
	}

	return &Client{
		apiKey:  apiKey,
		httpCli: httpCli,
	}
}

func (c *Client) Search(ctx context.Context, req *Request) (*Response, error) {
	u, err := url.Parse("https://api.marginalia.nu/")
	if err != nil {
		return nil, err
	}
	u.Path = "/" + c.apiKey + "/search/" + url.QueryEscape(req.Query)
	q := u.Query()
	if req.Count != nil {
		q.Set("count", fmt.Sprint(*req.Count))
	}
	if req.Index != nil {
		q.Set("index", fmt.Sprint(*req.Index))
	}
	u.RawQuery = q.Encode()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpCli.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
