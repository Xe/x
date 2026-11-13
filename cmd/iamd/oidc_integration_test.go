package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v3"
	"golang.org/x/net/html"
	"golang.org/x/oauth2"
	"within.website/x/cmd/iamd/models"
	provider "within.website/x/cmd/iamd/oidc"
)

type oidcTestNetwork struct{ hosts map[string]string }

func (n oidcTestNetwork) RoundTrip(r *http.Request) (*http.Response, error) {
	target, ok := n.hosts[r.URL.Host]
	if !ok {
		return nil, fmt.Errorf("unexpected network host: %s", r.URL.Host)
	}
	destination, _ := url.Parse(target)
	copy := r.Clone(r.Context())
	u := *r.URL
	u.Scheme = destination.Scheme
	u.Host = destination.Host
	copy.URL = &u
	return http.DefaultTransport.RoundTrip(copy)
}

// Exercise the real Go OIDC relying-party client through the generated Connect
// handlers and Vanguard, including both sides of the federated browser flow.
func TestOIDCRelyingPartyThroughVanguard(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "parent"))
	if err != nil {
		t.Fatal(err)
	}
	type grant struct{ nonce, challenge string }
	var mu sync.Mutex
	grants := map[string]grant{}
	parent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": "https://idp.xeserv.us", "authorization_endpoint": "https://idp.xeserv.us/authorize", "token_endpoint": "https://idp.xeserv.us/token", "jwks_uri": "https://idp.xeserv.us/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "parent", Algorithm: "RS256", Use: "sig"}}})
		case "/authorize":
			q := r.URL.Query()
			code := "test-upstream-code"
			mu.Lock()
			grants[code] = grant{nonce: q.Get("nonce"), challenge: q.Get("code_challenge")}
			mu.Unlock()
			target, _ := url.Parse(q.Get("redirect_uri"))
			values := target.Query()
			values.Set("code", code)
			values.Set("state", q.Get("state"))
			values.Set("iss", "https://idp.xeserv.us")
			target.RawQuery = values.Encode()
			http.Redirect(w, r, target.String(), 302)
		case "/token":
			_ = r.ParseForm()
			mu.Lock()
			g, ok := grants[r.Form.Get("code")]
			delete(grants, r.Form.Get("code"))
			mu.Unlock()
			hash := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if !ok || base64.RawURLEncoding.EncodeToString(hash[:]) != g.challenge {
				http.Error(w, "bad PKCE", 400)
				return
			}
			claims := map[string]any{"iss": "https://idp.xeserv.us", "sub": "upstream-subject", "aud": "iamd-parent", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(), "nonce": g.nonce, "name": "Alice", "email": "alice@example.com", "email_verified": true}
			data, _ := json.Marshal(claims)
			signed, _ := signer.Sign(data)
			token, _ := signed.CompactSerialize()
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "parent-access", "token_type": "Bearer", "id_token": token})
		default:
			http.NotFound(w, r)
		}
	}))
	defer parent.Close()
	network := &http.Client{Transport: oidcTestNetwork{hosts: map[string]string{"idp.xeserv.us": parent.URL}}}
	ctx := gooidc.ClientContext(context.Background(), network)
	dao, err := models.Open(filepath.Join(t.TempDir(), "iamd.db"))
	if err != nil {
		t.Fatal(err)
	}
	op, err := provider.New(ctx, dao, provider.Config{Issuer: "https://iamd.xeserv.us", UpstreamIssuer: "https://idp.xeserv.us", UpstreamClientID: "iamd-parent", UpstreamClientSecret: "parent-secret"})
	if err != nil {
		t.Fatal(err)
	}
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		op.ServeHTTP(w, r.WithContext(gooidc.ClientContext(r.Context(), network)))
	})
	h, err := newOIDCTransport(core, func(h http.Handler) http.Handler { return h })
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	transport := oidcTestNetwork{hosts: map[string]string{"idp.xeserv.us": parent.URL, "iamd.xeserv.us": server.URL}}
	network.Transport = transport
	rpCtx := gooidc.ClientContext(context.Background(), network)
	rp, err := gooidc.NewProvider(rpCtx, "https://iamd.xeserv.us")
	if err != nil {
		t.Fatal(err)
	}
	response, err := network.Post("https://iamd.xeserv.us/register", "application/json", strings.NewReader(`{"client_name":"Integration client","redirect_uris":["https://rp.example/callback"],"token_endpoint_auth_method":"none","grant_types":["authorization_code","refresh_token"],"scope":"openid profile email offline_access"}`))
	if err != nil {
		t.Fatal(err)
	}
	var registration map[string]any
	_ = json.NewDecoder(response.Body).Decode(&registration)
	response.Body.Close()
	if response.StatusCode != 201 {
		t.Fatalf("registration=%d %v", response.StatusCode, registration)
	}
	endpoint := rp.Endpoint()
	endpoint.AuthStyle = oauth2.AuthStyleInParams
	config := oauth2.Config{ClientID: registration["client_id"].(string), RedirectURL: "https://rp.example/callback", Endpoint: endpoint, Scopes: []string{"openid", "profile", "email", "offline_access"}}
	jar, _ := cookiejar.New(nil)
	browser := &http.Client{Transport: transport, Jar: jar, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Host == "rp.example" {
			return http.ErrUseLastResponse
		}
		if len(via) > 10 {
			return fmt.Errorf("redirect loop")
		}
		return nil
	}}
	verifier := oauth2.GenerateVerifier()
	nonce := "integration-nonce"
	state := "integration-state"
	authURL := config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), gooidc.Nonce(nonce))
	page, err := browser.Get(authURL)
	if err != nil {
		t.Fatal(err)
	}
	htmlBody, err := io.ReadAll(page.Body)
	page.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if page.StatusCode != 200 {
		t.Fatalf("browser status=%d %s", page.StatusCode, htmlBody)
	}
	fields := url.Values{}
	tree, err := html.Parse(strings.NewReader(string(htmlBody)))
	if err != nil {
		t.Fatal(err)
	}
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			name, value := "", ""
			for _, a := range n.Attr {
				if a.Key == "name" {
					name = a.Val
				}
				if a.Key == "value" {
					value = a.Val
				}
			}
			if name != "" {
				fields.Set(name, value)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(tree)
	if fields.Get("csrf") == "" {
		t.Fatalf("no consent form: %s", htmlBody)
	}
	fields.Set("decision", "approve")
	request, _ := http.NewRequest("POST", "https://iamd.xeserv.us/consent", strings.NewReader(fields.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://iamd.xeserv.us")
	approved, err := browser.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer approved.Body.Close()
	location, _ := url.Parse(approved.Header.Get("Location"))
	q := location.Query()
	if q.Get("code") == "" || q.Get("state") != state || q.Get("iss") != "https://iamd.xeserv.us" {
		body, _ := io.ReadAll(approved.Body)
		t.Fatalf("authorization=%d %s %s", approved.StatusCode, location, body)
	}
	tokens, err := config.Exchange(rpCtx, q.Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatal(err)
	}
	id, err := rp.Verifier(&gooidc.Config{ClientID: config.ClientID}).Verify(rpCtx, tokens.Extra("id_token").(string))
	if err != nil {
		t.Fatal(err)
	}
	if id.Nonce != nonce || id.Subject == "" {
		t.Fatal("ID token lost subject/nonce")
	}
	info, err := rp.UserInfo(rpCtx, oauth2.StaticTokenSource(tokens))
	if err != nil {
		t.Fatal(err)
	}
	if info.Subject != id.Subject || info.Email != "alice@example.com" {
		t.Fatalf("userinfo=%+v", info)
	}
	previousRefresh := tokens.RefreshToken
	tokens.Expiry = time.Now().Add(-time.Hour)
	refreshed, err := config.TokenSource(rpCtx, tokens).Token()
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.RefreshToken == "" || refreshed.RefreshToken == previousRefresh {
		t.Fatal("refresh token did not rotate")
	}
	replay, err := network.PostForm("https://iamd.xeserv.us/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {config.ClientID}, "refresh_token": {previousRefresh}})
	if err != nil {
		t.Fatal(err)
	}
	replay.Body.Close()
	if replay.StatusCode != 400 {
		t.Fatalf("refresh replay status=%d", replay.StatusCode)
	}
}
