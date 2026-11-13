package oidc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/authctx"
)

func TestCIMDAuthorizationAndLocalUserInfoRevocation(t *testing.T) {
	const clientID = "https://client.example/oauth/client.json"
	metadata := `{"client_id":"https://client.example/oauth/client.json","client_name":"URL client","redirect_uris":["https://client.example/cb"],"token_endpoint_auth_method":"none","scope":"openid profile"}`
	originalHTTP := defaultMetadataResolver.http
	defaultMetadataResolver.mu.Lock()
	originalCache := defaultMetadataResolver.cache
	defaultMetadataResolver.cache = nil
	defaultMetadataResolver.mu.Unlock()
	fetches := 0
	defaultMetadataResolver.http = &http.Client{Transport: metadataRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != clientID {
			t.Errorf("unexpected metadata fetch: %s", r.URL)
		}
		fetches++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}, "Cache-Control": {"no-store"}}, Body: io.NopCloser(strings.NewReader(metadata))}, nil
	})}
	t.Cleanup(func() {
		defaultMetadataResolver.http = originalHTTP
		defaultMetadataResolver.mu.Lock()
		defaultMetadataResolver.cache = originalCache
		defaultMetadataResolver.mu.Unlock()
	})
	s := testServer(t)
	user, err := s.store.dao.CreateUser(context.Background(), "Alice")
	if err != nil {
		t.Fatal(err)
	}
	raw := randomToken()
	browser := browserSession{Hash: hash(raw), UserUUID: user.UUID, CSRF: randomToken(), AuthTime: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	if err := s.store.db.Create(&browser).Error; err != nil {
		t.Fatal(err)
	}
	verifier := randomToken()
	q := url.Values{"client_id": {clientID}, "redirect_uri": {"https://client.example/cb"}, "response_type": {"code"}, "scope": {"openid profile"}, "state": {"test-state"}, "nonce": {"test-nonce"}, "code_challenge": {hash(verifier)}, "code_challenge_method": {"S256"}}
	get := httptest.NewRequest("GET", "/authorize?"+q.Encode(), nil)
	get.AddCookie(&http.Cookie{Name: sessionCookie, Value: raw})
	get.Header.Set("Accept", "application/json")
	consentView := httptest.NewRecorder()
	s.ServeHTTP(consentView, get)
	if consentView.Code != 200 {
		t.Fatalf("CIMD consent = %d: %s", consentView.Code, consentView.Body.String())
	}
	var view map[string]any
	if err := json.Unmarshal(consentView.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view["client_id"] != clientID || view["csrf"] != browser.CSRF {
		t.Fatal("CIMD consent identity incorrect")
	}
	q.Set("csrf", browser.CSRF)
	q.Set("decision", "approve")
	post := httptest.NewRequest("POST", "/consent", strings.NewReader(q.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: sessionCookie, Value: raw})
	authorized := httptest.NewRecorder()
	s.ServeHTTP(authorized, post)
	if authorized.Code != 303 {
		t.Fatalf("CIMD authorize = %d: %s", authorized.Code, authorized.Body.String())
	}
	location, err := url.Parse(authorized.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := location.Query().Get("code")
	if code == "" || location.Query().Get("iss") != s.cfg.Issuer {
		t.Fatal("CIMD authorization response missing code or issuer")
	}
	form := url.Values{"client_id": {clientID}, "grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {"https://client.example/cb"}, "code_verifier": {verifier}}
	tokenReq := httptest.NewRequest("POST", "/token", strings.NewReader(form.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokens := httptest.NewRecorder()
	s.ServeHTTP(tokens, tokenReq)
	if tokens.Code != 200 {
		t.Fatalf("CIMD token = %d: %s", tokens.Code, tokens.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(tokens.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	access := payload["access_token"].(string)
	userinfo := func(want int) {
		t.Helper()
		r := httptest.NewRequest("GET", "/userinfo", nil)
		r.Header.Set("Authorization", "Bearer "+access)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("CIMD UserInfo = %d: %s, want %d", w.Code, w.Body.String(), want)
		}
	}
	userinfo(200)
	if fetches < 4 {
		t.Fatalf("metadata was not revalidated across authorize and token: fetches=%d", fetches)
	}
	// Registry observation rows are optional for URL clients. Their immutable
	// grant binding and signed access token remain sufficient when no explicit
	// disable record exists.
	if err := s.store.db.Where("id = ?", clientID).Delete(&clientRecord{}).Error; err != nil {
		t.Fatal(err)
	}
	userinfo(200)
	disable := httptest.NewRequest("POST", "/admin/clients", strings.NewReader(`{"client_id":"`+clientID+`","disabled":true}`))
	disable.Header.Set("Content-Type", "application/json")
	disable = disable.WithContext(authctx.WithUser(disable.Context(), &iamv1.User{IsAdmin: true}))
	disabled := httptest.NewRecorder()
	s.ServeHTTP(disabled, disable)
	if disabled.Code != 200 {
		t.Fatalf("disable CIMD client = %d: %s", disabled.Code, disabled.Body.String())
	}
	userinfo(401)
}
