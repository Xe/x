package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	upstream "github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v3"
)

type federationTransport struct{ target *url.URL }

func (t federationTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	u := *r.URL
	u.Scheme = t.target.Scheme
	u.Host = t.target.Host
	copy.URL = &u
	return http.DefaultTransport.RoundTrip(copy)
}

func TestFederation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "parent"))
	if err != nil {
		t.Fatal(err)
	}
	sign := func(claims map[string]any) string {
		t.Helper()
		b, err := json.Marshal(claims)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := signer.Sign(b)
		if err != nil {
			t.Fatal(err)
		}
		token, err := sig.CompactSerialize()
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	nonce, challenge, scenario, profile := "", "", "valid", "Alice"
	parent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": "https://idp.xeserv.us", "authorization_endpoint": "https://idp.xeserv.us/authorize", "token_endpoint": "https://idp.xeserv.us/token", "jwks_uri": "https://idp.xeserv.us/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "parent", Algorithm: "RS256", Use: "sig"}}})
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			id, secret, ok := r.BasicAuth()
			if !ok || id != "upstream" || secret != "secret" {
				t.Errorf("upstream client auth missing")
			}
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
				t.Error("upstream PKCE verifier mismatch")
			}
			if r.Form.Get("redirect_uri") != "https://iam.example.com/upstream/callback" {
				t.Error("wrong upstream callback")
			}
			claims := map[string]any{"iss": "https://idp.xeserv.us", "sub": "parent-user", "aud": "upstream", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "auth_time": time.Now().Unix(), "nonce": nonce, "name": profile, "email": "alice@example.com", "email_verified": true}
			switch scenario {
			case "wrong nonce":
				claims["nonce"] = "other"
			case "wrong issuer":
				claims["iss"] = "https://evil.example"
			case "wrong audience":
				claims["aud"] = "other"
			case "wrong azp":
				claims["aud"] = []string{"upstream", "other"}
				claims["azp"] = "other"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "unused", "token_type": "Bearer", "id_token": sign(claims)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer parent.Close()
	target, _ := url.Parse(parent.URL)
	client := &http.Client{Transport: federationTransport{target: target}}
	ctx := upstream.ClientContext(context.Background(), client)
	st := testStore(t)
	s, err := newServer(ctx, st, Config{Issuer: "https://iam.example.com", UpstreamIssuer: "https://idp.xeserv.us", UpstreamClientID: "upstream", UpstreamClientSecret: "secret"}, true)
	if err != nil {
		t.Fatal(err)
	}
	login := func() (*http.Cookie, string) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "https://iam.example.com/login", nil).WithContext(ctx)
		s.ServeHTTP(w, r)
		if w.Code != 302 {
			t.Fatalf("login status=%d %s", w.Code, w.Body.String())
		}
		auth, err := url.Parse(w.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		q := auth.Query()
		nonce = q.Get("nonce")
		challenge = q.Get("code_challenge")
		if q.Get("code_challenge_method") != "S256" || nonce == "" {
			t.Fatal("missing nonce/PKCE")
		}
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == loginCookie {
				return cookie, q.Get("state")
			}
		}
		t.Fatal("missing login binding")
		return nil, ""
	}
	callback := func(cookie *http.Cookie, state string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", "https://iam.example.com/upstream/callback?"+url.Values{"state": {state}, "code": {"code"}, "iss": {"https://idp.xeserv.us"}}.Encode(), nil).WithContext(ctx)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	var sessionValue, userUUID string
	// Establish prerequisites outside subtests so individual cases can run alone.
	initialCookie, initialState := login()
	initial := callback(initialCookie, initialState)
	if initial.Code != http.StatusFound {
		t.Fatalf("initial login = %d", initial.Code)
	}
	for _, cookie := range initial.Result().Cookies() {
		if cookie.Name == sessionCookie {
			sessionValue = cookie.Value
		}
	}
	for _, tt := range []struct {
		name string
		want int
	}{{"wrong nonce", 400}, {"wrong issuer", 400}, {"wrong audience", 400}, {"wrong azp", 400}, {"valid", 302}} {
		t.Run(tt.name, func(t *testing.T) {
			scenario = tt.name
			cookie, state := login()
			w := callback(cookie, state)
			if w.Code != tt.want {
				t.Fatalf("callback status=%d want=%d body=%s", w.Code, tt.want, w.Body.String())
			}
			if tt.want == 302 {
				for _, cookie := range w.Result().Cookies() {
					if cookie.Name == sessionCookie {
						sessionValue = cookie.Value
					}
				}
			}
			if replay := callback(cookie, state); replay.Code != 400 {
				t.Errorf("state replay status=%d", replay.Code)
			}
		})
	}
	if sessionValue == "" {
		t.Fatal("missing local session")
	}
	var ident identity
	if err := st.db.First(&ident).Error; err != nil {
		t.Fatal(err)
	}
	userUUID = ident.UserUUID
	t.Run("upsert", func(t *testing.T) {
		scenario = "valid"
		profile = "Updated Alice"
		cookie, state := login()
		w := callback(cookie, state)
		if w.Code != 302 {
			t.Fatalf("upsert failed: %s", w.Body.String())
		}
		var n int64
		st.db.Model(&identity{}).Count(&n)
		if n != 1 {
			t.Fatalf("identities=%d", n)
		}
		user, err := st.dao.GetUser(ctx, userUUID)
		if err != nil || user.Name != profile || user.IsAdmin {
			t.Fatalf("profile/admin=%+v %v", user, err)
		}
		var keys int64
		st.db.Table("keys").Count(&keys)
		if keys != 0 {
			t.Fatal("login issued a signing key")
		}
	})
	t.Run("backchannel without exp", func(t *testing.T) {
		claims := map[string]any{"iss": "https://idp.xeserv.us", "sub": "parent-user", "aud": "upstream", "iat": time.Now().Unix(), "jti": "logout-once", "events": map[string]any{"http://schemas.openid.net/event/backchannel-logout": map[string]any{}}}
		claims["nonce"] = nil
		badToken := sign(claims)
		badRequest := httptest.NewRequest("POST", "https://iam.example.com/upstream/backchannel-logout", strings.NewReader(url.Values{"logout_token": {badToken}}.Encode())).WithContext(ctx)
		badRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rejected := httptest.NewRecorder()
		s.ServeHTTP(rejected, badRequest)
		if rejected.Code != http.StatusBadRequest {
			t.Errorf("logout token with nonce:null accepted: %d", rejected.Code)
		}
		delete(claims, "nonce")
		token := sign(claims)
		logout := func() *httptest.ResponseRecorder {
			r := httptest.NewRequest("POST", "https://iam.example.com/upstream/backchannel-logout", strings.NewReader(url.Values{"logout_token": {token}}.Encode())).WithContext(ctx)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			return w
		}
		w := logout()
		if w.Code != 200 {
			t.Fatalf("logout status=%d body=%s", w.Code, w.Body.String())
		}
		if logout().Code != 400 {
			t.Fatal("accepted logout replay")
		}
		var n int64
		st.db.Model(&browserSession{}).Where("revoked = ?", false).Count(&n)
		if n != 0 {
			t.Fatal("logout did not revoke sessions")
		}
	})
	t.Run("disabled account stays disabled", func(t *testing.T) {
		if err := st.dao.DisableUser(ctx, userUUID, "disabled"); err != nil {
			t.Fatal(err)
		}
		cookie, state := login()
		w := callback(cookie, state)
		if w.Code != 403 {
			t.Fatalf("disabled login status=%d", w.Code)
		}
	})
}

func TestConcurrentUpstreamIdentity(t *testing.T) {
	st := testStore(t)
	const count = 4
	errs := make(chan error, count)
	ids := make(chan string, count)
	start := make(chan struct{})
	for i := 0; i < count; i++ {
		go func() {
			<-start
			user, err := st.identityUser(context.Background(), "https://idp.xeserv.us", "concurrent-subject", "Alice", "", false)
			if err == nil {
				ids <- user.UUID
			}
			errs <- err
		}()
	}
	close(start)
	for i := 0; i < count; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent upsert: %v", err)
		}
	}
	close(ids)
	expected := ""
	for id := range ids {
		if expected == "" {
			expected = id
		}
		if id != expected {
			t.Error("concurrent login changed local UUID")
		}
	}
	var identities int64
	st.db.Model(&identity{}).Count(&identities)
	if identities != 1 {
		t.Fatalf("identities=%d", identities)
	}
}
