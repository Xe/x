package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	s := testStore(t)
	srv, err := newServer(context.Background(), s, Config{Issuer: "https://iam.example.com", UpstreamIssuer: "https://idp.xeserv.us"}, false)
	if err != nil {
		t.Fatal(err)
	}
	return srv
}
func TestDiscoveryAndKeyPersistence(t *testing.T) {
	s := testServer(t)
	for _, tt := range []struct{ name, path string }{{"oidc", "/.well-known/openid-configuration"}, {"oauth", "/.well-known/oauth-authorization-server"}, {"keys", "/jwks.json"}} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if w.Code != 200 {
				t.Fatalf("status = %d", w.Code)
			}
			var doc map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
		})
	}
	before := s.currentKey.ID
	again, err := newServer(context.Background(), s.store, s.cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	if before != again.currentKey.ID {
		t.Fatal("signing key did not survive restart")
	}
}
func TestAuthorizationRejectsUnsafeRedirectBeforeLogin(t *testing.T) {
	s := testServer(t)
	meta := ClientMetadata{ClientID: "client", RedirectURIs: []string{"https://client.example/cb"}, TokenEndpointAuthMethod: "none", Scope: "openid"}
	b, _ := json.Marshal(meta)
	if err := s.store.db.Create(&clientRecord{ID: "client", Metadata: b}).Error; err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ name, query string }{{"wrong redirect", "client_id=client&redirect_uri=https://evil.example/&response_type=code&scope=openid"}, {"missing pkce", "client_id=client&redirect_uri=https://client.example/cb&response_type=code&scope=openid"}, {"plain pkce", "client_id=client&redirect_uri=https://client.example/cb&response_type=code&scope=openid&code_challenge=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa&code_challenge_method=plain"}} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest("GET", "/authorize?"+tt.query, nil))
			if w.Code >= 300 && w.Code < 400 {
				t.Fatalf("invalid request redirected to %s", w.Header().Get("Location"))
			}
		})
	}
}

type tokenFixture struct {
	server                                   *Server
	tokens                                   map[string]any
	userID, browserHash, code, browserCookie string
	tokenRequest                             func(string) *httptest.ResponseRecorder
}

func issueTokens(t *testing.T) tokenFixture { return issueTokensForAudience(t, "") }
func issueTokensForAudience(t *testing.T, audience string) tokenFixture {
	t.Helper()
	s := testServer(t)
	ctx := context.Background()
	u, err := s.store.dao.CreateUser(ctx, "Alice")
	if err != nil {
		t.Fatal(err)
	}
	meta := ClientMetadata{ClientID: "client", RedirectURIs: []string{"https://client.example/cb"}, TokenEndpointAuthMethod: "none", Scope: "openid profile offline_access", GrantTypes: []string{"authorization_code", "refresh_token"}}
	if audience != "" {
		meta.Scope += " read"
		for _, target := range []string{audience, "https://other-api.example"} {
			if err := s.store.db.Create(&resource{Audience: target, Name: "API", Scopes: "read"}).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	b, _ := json.Marshal(meta)
	if err := s.store.db.Create(&clientRecord{ID: "client", Metadata: b}).Error; err != nil {
		t.Fatal(err)
	}
	raw := randomToken()
	browser := browserSession{Hash: hash(raw), UserUUID: u.UUID, CSRF: randomToken(), AuthTime: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	if err := s.store.db.Create(&browser).Error; err != nil {
		t.Fatal(err)
	}
	verifier := randomToken()
	q := url.Values{"client_id": {"client"}, "redirect_uri": {"https://client.example/cb"}, "response_type": {"code"}, "scope": {"openid profile offline_access"}, "state": {"test-state"}, "nonce": {"test-nonce"}, "code_challenge": {hash(verifier)}, "code_challenge_method": {"S256"}}
	if audience != "" {
		q.Set("resource", audience)
	} else {
		audience = s.cfg.Issuer + "/userinfo"
	}
	q.Set("scope", meta.Scope)
	// Explicit consent is persisted before completing authorization.
	if err := s.store.db.Create(&consent{UserUUID: u.UUID, ClientID: "client", Scopes: meta.Scope, Audience: audience, MetadataHash: hash(string(b))}).Error; err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/authorize?"+q.Encode(), nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: raw})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != 302 && w.Code != 303 {
		t.Fatalf("authorize = %d: %s", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := loc.Query().Get("code")
	if loc.Query().Get("iss") != s.cfg.Issuer {
		t.Fatal("authorization response missing issuer")
	}
	if code == "" {
		t.Fatalf("no code: %s", loc)
	}
	tokenRequest := func(code string) *httptest.ResponseRecorder {
		form := url.Values{"grant_type": {"authorization_code"}, "client_id": {"client"}, "code": {code}, "redirect_uri": {"https://client.example/cb"}, "code_verifier": {verifier}}
		r := httptest.NewRequest("POST", "/token", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		out := httptest.NewRecorder()
		s.ServeHTTP(out, r)
		return out
	}
	out := tokenRequest(code)
	if out.Code != 200 {
		t.Fatalf("token = %d: %s", out.Code, out.Body.String())
	}
	var tokens map[string]any
	_ = json.Unmarshal(out.Body.Bytes(), &tokens)
	if tokens["id_token"] == nil || tokens["refresh_token"] == nil {
		t.Fatalf("missing tokens: %v", tokens)
	}
	return tokenFixture{server: s, tokens: tokens, userID: u.UUID, browserHash: browser.Hash, code: code, browserCookie: raw, tokenRequest: tokenRequest}
}
func TestAuthorizationCodePKCEFlow(t *testing.T) {
	f := issueTokens(t)
	reused := f.tokenRequest(f.code)
	if reused.Code == 200 {
		t.Fatal("authorization code replay accepted")
	}
}
func refresh(t *testing.T, s *Server, token string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"grant_type": {"refresh_token"}, "client_id": {"client"}, "refresh_token": {token}}
	r := httptest.NewRequest("POST", "/token", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestAccessTokenClaimsAndUserInfo(t *testing.T) {
	f := issueTokens(t)
	s := f.server
	access := f.tokens["access_token"].(string)
	claims, err := s.verifyJWT(context.Background(), access, "at+JWT", s.cfg.Issuer+"/userinfo")
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != f.userID || claims["client_id"] != "client" {
		t.Fatalf("incorrect access claims: %v", claims)
	}
	for _, raw := range []string{access, f.tokens["id_token"].(string)} {
		r := httptest.NewRequest("GET", "/userinfo", nil)
		r.Header.Set("Authorization", "Bearer "+raw)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if raw == access && w.Code != 200 {
			t.Fatalf("userinfo = %d %s", w.Code, w.Body.String())
		}
		if raw != access && w.Code != 401 {
			t.Fatalf("ID token accepted by UserInfo: %d", w.Code)
		}
	}
	if err := s.rotateKey(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.verifyJWT(context.Background(), access, "at+JWT", s.cfg.Issuer+"/userinfo"); err != nil {
		t.Fatalf("rotation broke existing access token: %v", err)
	}
}
func TestRefreshRotationAndReuse(t *testing.T) {
	f := issueTokens(t)
	initial := f.tokens["refresh_token"].(string)
	first := refresh(t, f.server, initial)
	if first.Code != 200 {
		t.Fatalf("refresh = %d: %s", first.Code, first.Body.String())
	}
	var tokens map[string]any
	_ = json.Unmarshal(first.Body.Bytes(), &tokens)
	next := tokens["refresh_token"].(string)
	if next == initial {
		t.Fatal("refresh token did not rotate")
	}
	if got := refresh(t, f.server, initial); got.Code == 200 {
		t.Fatal("refresh replay accepted")
	}
	if got := refresh(t, f.server, next); got.Code == 200 {
		t.Fatal("replay did not revoke token family")
	}
}
func TestRefreshRevocationAndLifetime(t *testing.T) {
	for _, tt := range []struct {
		name   string
		revoke func(*testing.T, tokenFixture)
	}{
		{"disabled user", func(t *testing.T, f tokenFixture) {
			if err := f.server.store.dao.DisableUser(context.Background(), f.userID, "disabled"); err != nil {
				t.Fatal(err)
			}
		}},
		{"revoked browser", func(t *testing.T, f tokenFixture) {
			if err := f.server.store.db.Model(&browserSession{}).Where("hash = ?", f.browserHash).Update("revoked", true).Error; err != nil {
				t.Fatal(err)
			}
		}},
		{"revoked consent", func(t *testing.T, f tokenFixture) {
			if err := f.server.store.db.Model(&consent{}).Where("user_uuid = ?", f.userID).Update("revoked", true).Error; err != nil {
				t.Fatal(err)
			}
		}},
		{"idle expiry", func(t *testing.T, f tokenFixture) {
			if err := f.server.store.db.Model(&oauthRecord{}).Where("kind = ?", "refresh").Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
				t.Fatal(err)
			}
		}},
		{"metadata change", func(t *testing.T, f tokenFixture) {
			if err := f.server.store.db.Model(&clientRecord{}).Where("id = ?", "client").Update("metadata", []byte(`{"client_id":"client","redirect_uris":["https://client.example/other"],"token_endpoint_auth_method":"none","scope":"openid profile offline_access","grant_types":["authorization_code","refresh_token"]}`)).Error; err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := issueTokens(t)
			tt.revoke(t, f)
			if got := refresh(t, f.server, f.tokens["refresh_token"].(string)); got.Code == 200 {
				t.Fatal("revoked refresh token accepted")
			}
		})
	}
}
func TestConcurrentRefreshReuseRevokesFamily(t *testing.T) {
	f := issueTokens(t)
	raw := f.tokens["refresh_token"].(string)
	responses := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		go func() { responses <- refresh(t, f.server, raw) }()
	}
	first, second := <-responses, <-responses
	success := 0
	var issued string
	for _, out := range []*httptest.ResponseRecorder{first, second} {
		if out.Code == 200 {
			success++
			var tokens map[string]any
			_ = json.Unmarshal(out.Body.Bytes(), &tokens)
			issued = tokens["refresh_token"].(string)
		}
	}
	if success != 1 {
		t.Fatalf("successful concurrent refreshes = %d", success)
	}
	if out := refresh(t, f.server, issued); out.Code == 200 {
		t.Fatal("concurrent reuse left family active")
	}
}

func TestMissingBrowserCannotMintTokens(t *testing.T) {
	f := issueTokens(t)
	if err := f.server.store.db.Where("hash = ?", f.browserHash).Delete(&browserSession{}).Error; err != nil {
		t.Fatal(err)
	}
	if out := refresh(t, f.server, f.tokens["refresh_token"].(string)); out.Code == 200 {
		t.Fatal("token minted after browser binding disappeared")
	}
}
func TestResourceCatalogChangesBlockRefresh(t *testing.T) {
	f := issueTokensForAudience(t, "https://api.example")
	first := refresh(t, f.server, f.tokens["refresh_token"].(string))
	if first.Code != 200 {
		t.Fatalf("baseline resource refresh = %d: %s", first.Code, first.Body.String())
	}
	var tokens map[string]any
	_ = json.Unmarshal(first.Body.Bytes(), &tokens)
	if err := f.server.store.db.Model(&resource{}).Where("audience = ?", "https://api.example").Update("scopes", "write").Error; err != nil {
		t.Fatal(err)
	}
	if out := refresh(t, f.server, tokens["refresh_token"].(string)); out.Code == 200 {
		t.Fatal("removed resource scope remained valid because another resource still declared it")
	}
}
func TestNativeRedirectMatchesRawPathExceptPort(t *testing.T) {
	m := &ClientMetadata{ApplicationType: "native", RedirectURIs: []string{"http://127.0.0.1:1234/cb"}}
	for _, tt := range []struct {
		name, uri string
		want      bool
	}{{"port exception", "http://127.0.0.1:9876/cb", true}, {"encoded path", "http://127.0.0.1:9876/%63b", false}} {
		t.Run(tt.name, func(t *testing.T) {
			if got := validRedirect(m, tt.uri); got != tt.want {
				t.Fatalf("validRedirect = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestLogoutRequiresCSRFAndGetOnlyConfirms(t *testing.T) {
	f := issueTokens(t)
	var browser browserSession
	if err := f.server.store.db.Where("hash = ?", f.browserHash).First(&browser).Error; err != nil {
		t.Fatal(err)
	}
	// Full browser cookie is opaque; obtain a fresh cookie backed by the same row.
	raw := randomToken()
	if err := f.server.store.db.Model(&browserSession{}).Where("hash = ?", f.browserHash).Update("hash", hash(raw)).Error; err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, method, body string
		status             int
	}{{"confirm", "GET", "", 200}, {"missing CSRF", "POST", "", 403}, {"logout", "POST", "csrf=" + browser.CSRF, 200}} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/logout", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: raw})
			w := httptest.NewRecorder()
			f.server.ServeHTTP(w, r)
			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d", w.Code, tt.status)
			}
			var current browserSession
			if err := f.server.store.db.Where("hash = ?", hash(raw)).First(&current).Error; err != nil {
				t.Fatal(err)
			}
			if current.Revoked != (tt.name == "logout") {
				t.Fatalf("revoked = %v after %s", current.Revoked, tt.name)
			}
		})
	}
}

func TestPromptLoginContinuationSurvivesConsent(t *testing.T) {
	s := testServer(t)
	u, err := s.store.dao.CreateUser(context.Background(), "Alice")
	if err != nil {
		t.Fatal(err)
	}
	m := ClientMetadata{ClientID: "client", RedirectURIs: []string{"https://client.example/cb"}, Scope: "openid", TokenEndpointAuthMethod: "none"}
	encoded, _ := json.Marshal(m)
	if err := s.store.db.Create(&clientRecord{ID: "client", Metadata: encoded}).Error; err != nil {
		t.Fatal(err)
	}
	cookie := randomToken()
	browser := browserSession{Hash: hash(cookie), UserUUID: u.UUID, CSRF: randomToken(), AuthTime: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	if err := s.store.db.Create(&browser).Error; err != nil {
		t.Fatal(err)
	}
	q := url.Values{"client_id": {"client"}, "redirect_uri": {"https://client.example/cb"}, "response_type": {"code"}, "scope": {"openid"}, "state": {"test-state"}, "code_challenge": {hash(randomToken())}, "code_challenge_method": {"S256"}, "prompt": {"login"}}
	transaction := randomToken()
	marker := pendingLogin{Hash: hash(transaction), BrowserHash: browser.Hash, ReturnTo: "/authorize?" + q.Encode(), ExpiresAt: time.Now().Add(time.Minute)}
	if err := s.store.db.Create(&marker).Error; err != nil {
		t.Fatal(err)
	}
	q.Set("transaction", transaction)
	get := httptest.NewRequest("GET", "/authorize?"+q.Encode(), nil)
	get.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, get)
	if w.Code != 200 {
		t.Fatalf("consent get = %d: %s", w.Code, w.Body.String())
	}
	q.Set("csrf", browser.CSRF)
	q.Set("decision", "approve")
	post := httptest.NewRequest("POST", "/consent", strings.NewReader(q.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	out := httptest.NewRecorder()
	s.ServeHTTP(out, post)
	if out.Code != 303 {
		t.Fatalf("consent post = %d: %s", out.Code, out.Body.String())
	}
}

func TestAuthorizationErrorsIncludeIssuer(t *testing.T) {
	for _, tt := range []struct {
		name, decision string
		session        bool
	}{{"login_required", "", false}, {"consent_required", "", true}, {"access_denied", "deny", true}} {
		t.Run(tt.name, func(t *testing.T) {
			f := issueTokens(t)
			s := f.server
			if err := s.store.db.Where("user_uuid = ?", f.userID).Delete(&consent{}).Error; err != nil {
				t.Fatal(err)
			}
			q := url.Values{"client_id": {"client"}, "redirect_uri": {"https://client.example/cb"}, "response_type": {"code"}, "scope": {"openid"}, "state": {"test-state"}, "code_challenge": {hash(randomToken())}, "code_challenge_method": {"S256"}, "prompt": {"none"}}
			method := "GET"
			if tt.decision != "" {
				method = "POST"
				q.Set("decision", tt.decision)
				var b browserSession
				s.store.db.Where("hash = ?", f.browserHash).First(&b)
				q.Set("csrf", b.CSRF)
				q.Del("prompt")
			}
			var r *http.Request
			if method == "POST" {
				r = httptest.NewRequest(method, "/authorize", strings.NewReader(q.Encode()))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				r = httptest.NewRequest(method, "/authorize?"+q.Encode(), nil)
			}
			if tt.session {
				r.AddCookie(&http.Cookie{Name: sessionCookie, Value: f.browserCookie})
			}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			u, err := url.Parse(w.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			if u.Query().Get("error") != tt.name || u.Query().Get("iss") != s.cfg.Issuer {
				t.Fatalf("authorization error response = %s status=%d body=%s", u, w.Code, w.Body.String())
			}
		})
	}
}
func TestUserInfoHonorsLocalRevocation(t *testing.T) {
	for _, tt := range []struct {
		name   string
		revoke func(*testing.T, tokenFixture)
	}{
		{"access token", func(t *testing.T, f tokenFixture) {
			form := url.Values{"client_id": {"client"}, "token": {f.tokens["access_token"].(string)}, "token_type_hint": {"access_token"}}
			r := httptest.NewRequest("POST", "/revoke", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			f.server.ServeHTTP(w, r)
			if w.Code != 200 {
				t.Fatalf("revoke = %d", w.Code)
			}
		}},
		{"client", func(t *testing.T, f tokenFixture) {
			f.server.store.db.Model(&clientRecord{}).Where("id = ?", "client").Update("disabled", true)
		}},
		{"session", func(t *testing.T, f tokenFixture) {
			f.server.store.db.Model(&browserSession{}).Where("hash = ?", f.browserHash).Update("revoked", true)
		}},
		{"consent", func(t *testing.T, f tokenFixture) {
			f.server.store.db.Model(&consent{}).Where("user_uuid = ?", f.userID).Update("revoked", true)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := issueTokens(t)
			tt.revoke(t, f)
			r := httptest.NewRequest("GET", "/userinfo", nil)
			r.Header.Set("Authorization", "Bearer "+f.tokens["access_token"].(string))
			w := httptest.NewRecorder()
			f.server.ServeHTTP(w, r)
			if w.Code != 401 {
				t.Fatalf("revoked UserInfo = %d", w.Code)
			}
			if !strings.Contains(w.Header().Get("WWW-Authenticate"), `error="invalid_token"`) {
				t.Fatal("revoked bearer response lacks RFC6750 challenge")
			}
		})
	}
}
func TestBrowserNativeJSON(t *testing.T) {
	f := issueTokens(t)
	r := httptest.NewRequest("GET", "/account", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: f.browserCookie})
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	f.server.ServeHTTP(w, r)
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("native browser content type = %s", w.Header().Get("Content-Type"))
	}
	var doc map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["csrf"] == nil || doc["sessions"] == nil || doc["grants"] == nil {
		t.Fatal("native account lacks usable session data")
	}
}

func TestConcurrentSigningKeyRotation(t *testing.T) {
	s := testServer(t)
	done := make(chan error, 2)
	for range 2 {
		go func() { done <- s.rotateKey(context.Background()) }()
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	var rows []keyRecord
	if err := s.store.db.Where("retired_at IS NULL").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	s.keyMu.RLock()
	id := s.currentKey.ID
	s.keyMu.RUnlock()
	if len(rows) != 1 || rows[0].ID != id {
		t.Fatal("in-memory signing key differs from active SQL signing key")
	}
}

func TestExpiredSigningKeyRotatesBeforeTokenTransaction(t *testing.T) {
	f := issueTokens(t)
	s := f.server
	s.keyMu.Lock()
	oldID := s.currentKey.ID
	s.currentKey.CreatedAt = time.Now().Add(-31 * 24 * time.Hour)
	s.keyMu.Unlock()
	if err := s.store.db.Model(&keyRecord{}).Where("id = ?", oldID).Update("created_at", time.Now().Add(-31*24*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() { result <- refresh(t, s, f.tokens["refresh_token"].(string)) }()
	select {
	case out := <-result:
		if out.Code != 200 {
			t.Fatalf("rotated refresh = %d: %s", out.Code, out.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("expired signing key deadlocked SQLite token transaction")
	}
	s.keyMu.RLock()
	currentID := s.currentKey.ID
	s.keyMu.RUnlock()
	if currentID == oldID {
		t.Fatal("expired signing key did not rotate")
	}
}
