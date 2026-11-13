package oidc

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v3"
)

func TestValidateMetadata(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mutate  func(*ClientMetadata)
		url     bool
		wantErr bool
	}{
		{name: "confidential DCR defaults"},
		{name: "HTTPS CIMD", url: true},
		{name: "root CIMD", url: true, mutate: func(m *ClientMetadata) { m.ClientID = "https://example.com/" }, wantErr: true},
		{name: "dot path", url: true, mutate: func(m *ClientMetadata) { m.ClientID = "https://example.com/a/%2e%2e/b" }, wantErr: true},
		{name: "secret CIMD", url: true, mutate: func(m *ClientMetadata) { m.TokenEndpointAuthMethod = "client_secret_basic" }, wantErr: true},
		{name: "confidential DCR", mutate: func(m *ClientMetadata) { m.TokenEndpointAuthMethod = "client_secret_basic" }},
		{name: "HTTP callback", mutate: func(m *ClientMetadata) { m.RedirectURIs = []string{"http://example.com/callback"} }, wantErr: true},
		{name: "native loopback", mutate: func(m *ClientMetadata) {
			m.ApplicationType = "native"
			m.RedirectURIs = []string{"http://127.0.0.1:1234/callback"}
		}},
		{name: "fragment callback", mutate: func(m *ClientMetadata) { m.RedirectURIs = []string{"https://example.com/cb#oops"} }, wantErr: true},
		{name: "unsupported grant", mutate: func(m *ClientMetadata) { m.GrantTypes = []string{"password"} }, wantErr: true},
		{name: "CIMD grant intersection", url: true, mutate: func(m *ClientMetadata) { m.GrantTypes = []string{"password", "authorization_code"} }},
		{name: "empty supported auth", url: true, mutate: func(m *ClientMetadata) { m.TokenEndpointAuthMethodsSupported = []string{} }, wantErr: true},
		{name: "inconsistent auth", url: true, mutate: func(m *ClientMetadata) {
			m.TokenEndpointAuthMethod = "none"
			m.TokenEndpointAuthMethodsSupported = []string{"private_key_jwt"}
		}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &ClientMetadata{ClientID: "https://example.com/client.json", RedirectURIs: []string{"https://example.com/callback"}}
			if tt.mutate != nil {
				tt.mutate(m)
			}
			err := ValidateMetadata(m, tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateMetadata error = %v, want error %v", err, tt.wantErr)
			}
			if err == nil && m.TokenEndpointAuthMethod == "" {
				t.Error("missing explicit authentication default")
			}
		})
	}
}

func TestPublicMetadataAddress(t *testing.T) {
	for _, tt := range []struct {
		name, ip string
		want     bool
	}{
		{"public", "8.8.8.8", true}, {"public v6", "2606:4700:4700::1111", true}, {"loopback", "127.0.0.1", false}, {"private", "10.0.0.1", false}, {"mapped", "::ffff:127.0.0.1", false}, {"link local", "169.254.169.254", false}, {"CGNAT", "100.64.0.1", false}, {"benchmark", "198.18.0.1", false}, {"documentation", "192.0.2.1", false}, {"NAT64", "64:ff9b::7f00:1", false}, {"multicast", "224.0.0.1", false}, {"v6 private", "fd00::1", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := publicMetadataIP(net.ParseIP(tt.ip)); got != tt.want {
				t.Errorf("publicMetadataIP(%s)=%v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

type metadataRoundTrip func(*http.Request) (*http.Response, error)

func (f metadataRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestMetadataFetch(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		wantErr    bool
	}{
		{"valid", `{"client_id":"https://example.com/client.json","redirect_uris":["https://example.com/cb"]}`, 200, false},
		{"mismatch", `{"client_id":"https://example.com:443/client.json","redirect_uris":["https://example.com/cb"]}`, 200, true},
		{"redirect", `{}`, 302, true}, {"array", `[]`, 200, true}, {"null", `null`, 200, true}, {"trailing", `{} {}`, 200, true},
		{"oversize", strings.Repeat(" ", 65537), 200, true}, {"secret", `{"client_secret":"secret"}`, 200, true}, {"secret expiry", `{"client_secret_expires_at":0}`, 200, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := metadataResolver{http: &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Accept") != "application/json" {
					t.Error("missing JSON Accept header")
				}
				return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body)), Header: make(http.Header)}, nil
			})}}
			_, err := r.resolve(context.Background(), "https://example.com/client.json")
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolve error = %v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataPinnedDial(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ips      []net.IPAddr
		wantDial bool
	}{
		{name: "pins public DNS", ips: []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, wantDial: true},
		{name: "private DNS blocked", ips: []net.IPAddr{{IP: net.ParseIP("10.1.2.3")}}},
		{name: "mixed DNS blocked", ips: []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("127.0.0.1")}}},
		{name: "empty DNS blocked"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lookups, dials := 0, 0
			client := newMetadataHTTPClient(func(ctx context.Context, host string) ([]net.IPAddr, error) {
				lookups++
				if host != "example.com" {
					t.Errorf("lookup host=%q", host)
				}
				return tt.ips, nil
			}, func(ctx context.Context, network, address string) (net.Conn, error) {
				dials++
				if address != "8.8.8.8:443" {
					t.Errorf("dial address=%q, want pinned public address", address)
				}
				return nil, errors.New("test stops before connecting")
			})
			_, err := client.Get("https://example.com/client.json")
			if err == nil {
				t.Error("expected connection failure")
			}
			if lookups != 1 {
				t.Errorf("lookups=%d, want 1", lookups)
			}
			if (dials > 0) != tt.wantDial {
				t.Errorf("dials=%d, want dial %v", dials, tt.wantDial)
			}
		})
	}
}

func TestMetadataRejectsKeyMaterial(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public := &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, Use: "sig"}}}
	for _, tt := range []struct {
		name    string
		keys    *jose.JSONWebKeySet
		wantErr bool
	}{
		{name: "public RSA", keys: public},
		{name: "private RSA", keys: &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: key}}}, wantErr: true},
		{name: "symmetric", keys: &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: []byte("a sufficiently long symmetric key")}}}, wantErr: true},
		{name: "empty", keys: &jose.JSONWebKeySet{}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.keys)
			if err != nil {
				t.Fatal(err)
			}
			var m ClientMetadata
			err = json.Unmarshal(append(append([]byte(`{"jwks":`), raw...), '}'), &m)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshal error=%v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestHydrateKeys(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey}}})
	if err != nil {
		t.Fatal(err)
	}
	r := metadataResolver{http: &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
	})}}
	m := &ClientMetadata{JWKSURI: "https://example.com/keys.json"}
	if err := r.hydrate(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	if m.JWKSURI != "" || m.JWKS == nil {
		t.Fatal("hydration must replace remote URI with validated keys")
	}
	client := MetadataClient(m)
	if client.JSONWebKeysURI != "" || client.JSONWebKeys == nil {
		t.Fatal("Fosite must receive keys without an unrestricted remote URI")
	}
}

func TestMetadataDoesNotCache(t *testing.T) {
	calls := 0
	r := metadataResolver{http: &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"client_id":"https://example.com/client.json","redirect_uris":["https://example.com/cb"]}`)), Header: http.Header{"Cache-Control": []string{"no-store"}}}, nil
	})}}
	for range 2 {
		if _, err := r.resolve(context.Background(), "https://example.com/client.json"); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Errorf("requests=%d, want 2", calls)
	}
}

func TestMetadataRedirectsRejected(t *testing.T) {
	calls := 0
	client := newMetadataHTTPClient(nil, nil)
	client.Transport = metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://elsewhere.example/client.json"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})
	r := metadataResolver{http: client}
	if _, err := r.resolve(context.Background(), "https://example.com/client.json"); err == nil {
		t.Fatal("redirect was accepted")
	}
	if calls != 1 {
		t.Errorf("requests=%d, want 1", calls)
	}
}

func TestMetadataCacheTTL(t *testing.T) {
	for _, tt := range []struct {
		name, control, age string
		want               time.Duration
	}{
		{name: "default", want: 5 * time.Minute}, {name: "capped", control: "max-age=7200", want: time.Hour},
		{name: "explicit", control: "max-age=10", want: 10 * time.Second}, {name: "private no-store", control: "private, max-age=60, no-store", want: 0},
		{name: "revalidate", control: "no-cache", want: 0}, {name: "zero", control: "max-age=0", want: 0},
		{name: "bad age", control: "max-age=bad", want: 0}, {name: "aged", control: "max-age=10", age: "3", want: 7 * time.Second},
		{name: "stale", control: "max-age=10", age: "11", want: 0}, {name: "overflow", control: "max-age=9223372036854775807", want: time.Hour},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := metadataCacheTTL(http.Header{"Cache-Control": []string{tt.control}, "Age": []string{tt.age}})
			if got != tt.want {
				t.Errorf("TTL=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadataCache(t *testing.T) {
	calls := 0
	r := metadataResolver{http: &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		body := fmt.Sprintf(`{"client_id":%q,"redirect_uris":["https://example.com/cb"]}`, req.URL.String())
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	first, err := r.resolve(context.Background(), "https://example.com/client.json")
	if err != nil {
		t.Fatal(err)
	}
	first.RedirectURIs[0] = "https://attacker.example/cb"
	second, err := r.resolve(context.Background(), "https://example.com/client.json")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || second.RedirectURIs[0] != "https://example.com/cb" {
		t.Fatal("cache did not isolate caller mutations")
	}
	r.mu.Lock()
	entry := r.cache[first.ClientID]
	entry.expires = time.Now().Add(-time.Second)
	r.cache[first.ClientID] = entry
	r.mu.Unlock()
	if _, err := r.resolve(context.Background(), first.ClientID); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("calls=%d, want 2 after expiry", calls)
	}
	for i := range 300 {
		if _, err := r.resolve(context.Background(), fmt.Sprintf("https://example.com/%d.json", i)); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.cache) > 256 {
		t.Errorf("cache has %d entries", len(r.cache))
	}
}

func TestMetadataInvalidNotCached(t *testing.T) {
	calls := 0
	r := metadataResolver{http: &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"client_id":"https://wrong.example/id"}`)), Header: make(http.Header)}, nil
	})}}
	for range 2 {
		if _, err := r.resolve(context.Background(), "https://example.com/client.json"); err == nil {
			t.Fatal("accepted invalid metadata")
		}
	}
	if calls != 2 || len(r.cache) != 0 {
		t.Fatal("invalid metadata was cached")
	}
}

func TestMetadataKeyCachePolicy(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, control string
		wantCalls     int
	}{
		{name: "keys cached briefly", control: "max-age=10", wantCalls: 2},
		{name: "keys forbid storage", control: "no-store", wantCalls: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			r := metadataResolver{http: &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
				calls++
				body := []byte(`{"client_id":"https://example.com/client.json","redirect_uris":["https://example.com/cb"],"token_endpoint_auth_methods_supported":["none","private_key_jwt"],"jwks_uri":"https://keys.example/keys.json"}`)
				control := "max-age=3600"
				if req.URL.Hostname() == "keys.example" {
					body = keys
					control = tt.control
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{"Cache-Control": []string{control}}}, nil
			})}}
			for range 2 {
				m, err := r.resolve(context.Background(), "https://example.com/client.json")
				if err != nil {
					t.Fatal(err)
				}
				if m.TokenEndpointAuthMethod != "private_key_jwt" {
					t.Error("must prefer advertised public-key authentication")
				}
			}
			if calls != tt.wantCalls {
				t.Errorf("calls=%d, want %d", calls, tt.wantCalls)
			}
			if entry, ok := r.cache["https://example.com/client.json"]; ok && time.Until(entry.expires) > 10*time.Second {
				t.Fatal("metadata cache outlives keys")
			}
		})
	}
}

func TestMetadataConcurrentCache(t *testing.T) {
	r := metadataResolver{http: &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"client_id":"https://example.com/client.json","redirect_uris":["https://example.com/cb"]}`)), Header: make(http.Header)}, nil
	})}}
	var wg sync.WaitGroup
	for range 64 {
		wg.Go(func() {
			m, err := r.resolve(context.Background(), "https://example.com/client.json")
			if err != nil {
				t.Error(err)
				return
			}
			m.RedirectURIs[0] = "https://changed.example/cb"
		})
	}
	wg.Wait()
}

func TestMetadataAuthenticationDefaults(t *testing.T) {
	for _, tt := range []struct {
		name      string
		urlClient bool
		explicit  string
		want      string
	}{
		{name: "DCR defaults confidential", want: "client_secret_basic"},
		{name: "CIMD defaults public", urlClient: true, want: "none"},
		{name: "DCR explicit public", explicit: "none", want: "none"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &ClientMetadata{ClientID: "https://example.com/client.json", RedirectURIs: []string{"https://example.com/cb"}, TokenEndpointAuthMethod: tt.explicit}
			if err := ValidateMetadata(m, tt.urlClient); err != nil {
				t.Fatal(err)
			}
			if m.TokenEndpointAuthMethod != tt.want {
				t.Errorf("auth=%q, want %q", m.TokenEndpointAuthMethod, tt.want)
			}
		})
	}
}
func TestMetadataNativeCallbacks(t *testing.T) {
	for _, tt := range []struct {
		name, application, uri string
		wantErr                bool
	}{
		{name: "private-use native", application: "native", uri: "com.example.app:/oauth2redirect"},
		{name: "private-use web denied", application: "web", uri: "com.example.app:/oauth2redirect", wantErr: true},
		{name: "opaque denied", application: "native", uri: "com.example.app:oauth2redirect", wantErr: true},
		{name: "non-reverse-domain denied", application: "native", uri: "myapp:/oauth2redirect", wantErr: true},
		{name: "fragment denied", application: "native", uri: "com.example.app:/oauth2redirect#fragment", wantErr: true},
		{name: "script denied", application: "native", uri: "javascript:alert(1)", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &ClientMetadata{ClientID: "registered", RedirectURIs: []string{tt.uri}, ApplicationType: tt.application}
			err := ValidateMetadata(m, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("error=%v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataAssertionAlgorithms(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ec384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	edKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, alg, keyAlg string
		key               any
		wantErr           bool
	}{
		{name: "RSA default", key: &rsaKey.PublicKey},
		{name: "explicit RSA", alg: "RS256", key: &rsaKey.PublicKey},
		{name: "explicit EC", alg: "ES256", key: &ecKey.PublicKey},
		{name: "EC cannot satisfy default RSA", key: &ecKey.PublicKey, wantErr: true},
		{name: "RSA cannot satisfy EC", alg: "ES256", key: &rsaKey.PublicKey, wantErr: true},
		{name: "EC cannot satisfy RSA", alg: "RS256", key: &ecKey.PublicKey, wantErr: true},
		{name: "EdDSA unavailable", alg: "EdDSA", key: edKey, wantErr: true},
		{name: "RSA384 unavailable", alg: "RS384", key: &rsaKey.PublicKey, wantErr: true},
		{name: "PSS unavailable", alg: "PS256", key: &rsaKey.PublicKey, wantErr: true},
		{name: "wrong curve", alg: "ES256", key: &ec384.PublicKey, wantErr: true},
		{name: "JWK algorithm mismatch", alg: "RS256", keyAlg: "ES256", key: &rsaKey.PublicKey, wantErr: true},
		{name: "JWK unsupported algorithm", alg: "RS256", keyAlg: "RS384", key: &rsaKey.PublicKey, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &ClientMetadata{ClientID: "registered", RedirectURIs: []string{"https://example.com/cb"}, TokenEndpointAuthMethod: "private_key_jwt", TokenEndpointAuthSigningAlg: tt.alg, JWKS: &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: tt.key, Algorithm: tt.keyAlg}}}}
			err := ValidateMetadata(m, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("error=%v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataHydrationChecksSelectedAlgorithm(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, Algorithm: "ES256"}}})
	if err != nil {
		t.Fatal(err)
	}
	r := metadataResolver{http: &http.Client{Transport: metadataRoundTrip(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(keys)), Header: make(http.Header)}, nil
	})}}
	m := &ClientMetadata{ClientID: "registered", RedirectURIs: []string{"https://example.com/cb"}, TokenEndpointAuthMethod: "private_key_jwt", JWKSURI: "https://example.com/keys.json"}
	if err := ValidateMetadata(m, false); err != nil {
		t.Fatal(err)
	}
	if err := r.hydrate(context.Background(), m); err == nil {
		t.Error("accepted EC keys for default RS256 client assertion algorithm")
	}
}
