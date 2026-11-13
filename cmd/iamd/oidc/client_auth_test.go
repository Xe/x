package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v3"
)

func TestRegisteredClientAuthentication(t *testing.T) {
	for _, method := range []string{"none", "client_secret_basic", "private_key_jwt"} {
		t.Run(method, func(t *testing.T) {
			s := testServer(t)
			m := ClientMetadata{RedirectURIs: []string{"https://client.example/cb"}, Scope: "openid", TokenEndpointAuthMethod: method}
			var private *rsa.PrivateKey
			if method == "private_key_jwt" {
				var err error
				private, err = rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					t.Fatal(err)
				}
				m.JWKS = &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &private.PublicKey, KeyID: "client-key", Use: "sig", Algorithm: "RS256"}}}
				m.TokenEndpointAuthSigningAlg = "RS256"
			}
			encoded, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/register", strings.NewReader(string(encoded)))
			r.Header.Set("Content-Type", "application/json")
			registered := httptest.NewRecorder()
			s.ServeHTTP(registered, r)
			if registered.Code != 201 {
				t.Fatalf("registration = %d: %s", registered.Code, registered.Body.String())
			}
			var response map[string]any
			if err := json.Unmarshal(registered.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			id := response["client_id"].(string)
			var assertion string
			if method == "private_key_jwt" {
				claims := map[string]any{"iss": id, "sub": id, "aud": s.cfg.Issuer + "/token", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(), "jti": randomToken()}
				payload, _ := json.Marshal(claims)
				signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: private}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "client-key"))
				if err != nil {
					t.Fatal(err)
				}
				signed, err := signer.Sign(payload)
				if err != nil {
					t.Fatal(err)
				}
				assertion, err = signed.CompactSerialize()
				if err != nil {
					t.Fatal(err)
				}
			}
			request := func(bad bool) *httptest.ResponseRecorder {
				form := url.Values{"grant_type": {"authorization_code"}, "client_id": {id}, "code": {"unknown-code"}, "redirect_uri": {"https://client.example/cb"}, "code_verifier": {randomToken()}}
				if assertion != "" {
					form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
					form.Set("client_assertion", assertion)
				}
				r := httptest.NewRequest("POST", "/token", strings.NewReader(form.Encode()))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				if method == "client_secret_basic" {
					secret := response["client_secret"].(string)
					if bad {
						secret = "invalid"
					}
					r.SetBasicAuth(id, secret)
				}
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				return w
			}
			out := request(false)
			var problem map[string]any
			_ = json.Unmarshal(out.Body.Bytes(), &problem)
			if problem["error"] != "invalid_grant" {
				t.Fatalf("client did not authenticate: %d %s", out.Code, out.Body.String())
			}
			if method != "none" {
				again := request(true)
				var problem map[string]any
				_ = json.Unmarshal(again.Body.Bytes(), &problem)
				if problem["error"] != "invalid_client" {
					t.Fatalf("bad secret/replayed assertion = %d: %s", again.Code, again.Body.String())
				}
			}
		})
	}
}
