package mastodon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"within.website/x/web"
)

type CreateApplicationRequest struct {
	ClientName   string `json:"client_name"`
	RedirectURIs string `json:"redirect_uris"`
	Scopes       string `json:"scopes"`
	Website      string `json:"website"`
}

type OAuth2Application struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Website      string `json:"website"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	VapidKey     string `json:"vapid_key"`
}

type TokenInfo struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	Scope       string       `json:"scope"`
	CreatedAt   MastodonDate `json:"created_at"`
}

func (c *Client) CreateApplication(ctx context.Context, car CreateApplicationRequest) (*OAuth2Application, error) {
	if car.RedirectURIs == "" {
		car.RedirectURIs = "urn:ietf:wg:oauth:2.0:oob"
	}

	if car.Scopes == "" {
		car.Scopes = "read write follow push"
	}

	resp, err := c.doJSONPost(ctx, "/api/v1/apps", http.StatusOK, car)
	if err != nil {
		return nil, err
	}

	var result OAuth2Application
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) AuthorizeURL(app *OAuth2Application, scope string) (string, error) {
	u, err := c.server.Parse("/oauth/authorize")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("client_id", app.ClientID)
	q.Set("scope", scope)
	q.Set("redirect_uri", app.RedirectURI)
	q.Set("response_type", "code")

	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (c *Client) FetchToken(ctx context.Context, app *OAuth2Application, code, scope string) (*TokenInfo, error) {
	u, err := c.server.Parse("/oauth/token")
	if err != nil {
		return nil, err
	}

	form := url.Values{}

	form.Set("client_id", app.ClientID)
	form.Set("client_secret", app.ClientSecret)
	form.Set("redirect_uri", app.RedirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("scope", scope)

	resp, err := c.cli.PostForm(u.String(), form)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	defer resp.Body.Close()

	var result TokenInfo
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) VerifyCredentials(ctx context.Context) error {
	h := http.Header{}
	h.Set("Accept", "application/json")
	_, err := c.doRequest(ctx, http.MethodGet, "/api/v1/apps/verify_credentials", h, http.StatusOK, nil)
	return err
}
