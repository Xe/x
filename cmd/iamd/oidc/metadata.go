package oidc

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/ory/fosite"
)

// ClientMetadata contains the registration metadata understood by iamd.
// CIMD processing follows draft-ietf-oauth-client-id-metadata-document-03:
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-03
// Informational URLs are deliberately neither fetched nor interpreted.
type ClientMetadata struct {
	ClientID                          string              `json:"client_id"`
	ClientName                        string              `json:"client_name"`
	RedirectURIs                      []string            `json:"redirect_uris"`
	GrantTypes                        []string            `json:"grant_types"`
	ResponseTypes                     []string            `json:"response_types"`
	TokenEndpointAuthMethod           string              `json:"token_endpoint_auth_method"`
	TokenEndpointAuthMethodsSupported []string            `json:"token_endpoint_auth_methods_supported,omitempty"`
	Scope                             string              `json:"scope"`
	ApplicationType                   string              `json:"application_type"`
	JWKS                              *jose.JSONWebKeySet `json:"jwks,omitempty"`
	JWKSURI                           string              `json:"jwks_uri,omitempty"`
	TokenEndpointAuthSigningAlg       string              `json:"token_endpoint_auth_signing_alg,omitempty"`
}

func (m *ClientMetadata) UnmarshalJSON(data []byte) error {
	type metadata ClientMetadata
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return errors.New("metadata must be a JSON object")
	}
	for _, key := range []string{"client_secret", "client_secret_expires_at"} {
		if _, ok := fields[key]; ok {
			return fmt.Errorf("metadata must not contain %s", key)
		}
	}
	if _, hasKeys := fields["jwks"]; hasKeys {
		if _, hasURI := fields["jwks_uri"]; hasURI {
			return errors.New("jwks and jwks_uri are mutually exclusive")
		}
	}
	if auth, present := fields["token_endpoint_auth_methods_supported"]; present && bytes.Equal(bytes.TrimSpace(auth), []byte("null")) {
		return errors.New("token_endpoint_auth_methods_supported must be an array")
	}
	var decoded metadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if err := validatePublicKeys(decoded.JWKS); err != nil {
		return err
	}
	*m = ClientMetadata(decoded)
	return nil
}

// ValidateMetadata checks and normalizes metadata before storing or using it.
func ValidateMetadata(m *ClientMetadata, urlClient bool) error {
	if m == nil || m.ClientID == "" {
		return errors.New("client_id is required")
	}
	if urlClient {
		if _, err := metadataURL(m.ClientID, true); err != nil {
			return err
		}
	}
	if m.ApplicationType == "" {
		m.ApplicationType = "web"
	}
	if !slices.Contains([]string{"web", "native", "spa"}, m.ApplicationType) {
		return errors.New("unsupported application_type")
	}
	if len(m.RedirectURIs) == 0 {
		return errors.New("redirect_uris is required")
	}
	for _, raw := range m.RedirectURIs {
		u, err := url.Parse(raw)
		if err != nil || u.User != nil || u.Fragment != "" || strings.Contains(raw, "#") || strings.ContainsAny(raw, "\\*\r\n") {
			return errors.New("invalid redirect URI")
		}
		if m.ApplicationType == "native" && strings.Contains(u.Scheme, ".") && u.Opaque == "" && u.Path != "" {
			continue
		}
		if u.Host == "" {
			return errors.New("redirect URI requires a host")
		}
		if u.Scheme == "https" {
			continue
		}
		ip := net.ParseIP(u.Hostname())
		if m.ApplicationType != "native" || u.Scheme != "http" || ip == nil || !ip.IsLoopback() {
			return errors.New("redirect URI must use HTTPS, or native HTTP loopback")
		}
	}
	var err error
	m.GrantTypes, err = metadataValues(m.GrantTypes, []string{"authorization_code"}, []string{"authorization_code", "refresh_token"}, urlClient)
	if err != nil {
		return fmt.Errorf("grant_types: %w", err)
	}
	m.ResponseTypes, err = metadataValues(m.ResponseTypes, []string{"code"}, []string{"code"}, urlClient)
	if err != nil {
		return fmt.Errorf("response_types: %w", err)
	}
	methods := m.TokenEndpointAuthMethodsSupported
	if methods != nil {
		if len(methods) == 0 {
			return errors.New("empty token_endpoint_auth_methods_supported")
		}
		if m.TokenEndpointAuthMethod != "" && !slices.Contains(methods, m.TokenEndpointAuthMethod) {
			return errors.New("inconsistent token endpoint authentication methods")
		}
		if urlClient {
			for _, method := range methods {
				if strings.HasPrefix(method, "client_secret_") {
					return errors.New("CIMD cannot use shared secrets")
				}
			}
		}
		switch {
		case slices.Contains(methods, "private_key_jwt"):
			m.TokenEndpointAuthMethod = "private_key_jwt"
		case slices.Contains(methods, "none"):
			m.TokenEndpointAuthMethod = "none"
		case !urlClient && slices.Contains(methods, "client_secret_basic"):
			m.TokenEndpointAuthMethod = "client_secret_basic"
		default:
			return errors.New("no supported token endpoint authentication method")
		}
	}
	if m.TokenEndpointAuthMethod == "" {
		if urlClient {
			m.TokenEndpointAuthMethod = "none"
		} else {
			m.TokenEndpointAuthMethod = "client_secret_basic"
		}
	}
	switch m.TokenEndpointAuthMethod {
	case "none":
	case "client_secret_basic":
		if urlClient {
			return errors.New("CIMD cannot use shared secrets")
		}
	case "private_key_jwt":
		if m.JWKS == nil && m.JWKSURI == "" {
			return errors.New("private_key_jwt requires public keys")
		}
	default:
		return errors.New("unsupported token_endpoint_auth_method")
	}
	if m.JWKS != nil && m.JWKSURI != "" {
		return errors.New("jwks and jwks_uri are mutually exclusive")
	}
	if m.JWKSURI != "" {
		if _, err := metadataURL(m.JWKSURI, false); err != nil {
			return fmt.Errorf("jwks_uri: %w", err)
		}
	}
	if err := validatePublicKeys(m.JWKS); err != nil {
		return err
	}
	if m.TokenEndpointAuthSigningAlg != "" && !slices.Contains([]string{"RS256", "ES256"}, m.TokenEndpointAuthSigningAlg) {
		return errors.New("unsupported client assertion signing algorithm")
	}
	return validateAssertionKeys(m)
}

func metadataValues(values, defaults, allowed []string, filter bool) ([]string, error) {
	if values == nil {
		return slices.Clone(defaults), nil
	}
	var result []string
	for _, value := range values {
		if slices.Contains(allowed, value) {
			if !slices.Contains(result, value) {
				result = append(result, value)
			}
		} else if !filter {
			return nil, errors.New("unsupported value")
		}
	}
	if len(result) == 0 {
		return nil, errors.New("no supported values")
	}
	return result, nil
}

func validatePublicKeys(keys *jose.JSONWebKeySet) error {
	if keys == nil {
		return nil
	}
	if len(keys.Keys) == 0 {
		return errors.New("empty jwks")
	}
	for _, key := range keys.Keys {
		if !key.IsPublic() || !key.Valid() {
			return errors.New("jwks must contain valid asymmetric public keys")
		}
		if key.Use != "" && key.Use != "sig" {
			return errors.New("jwks must contain signing keys")
		}
		switch k := key.Key.(type) {
		case *rsa.PublicKey:
			if k.N.BitLen() < 2048 {
				return errors.New("RSA key must have at least 2048 bits")
			}
			if key.Algorithm != "" && key.Algorithm != "RS256" {
				return errors.New("RSA signing keys must use RS256")
			}
		case *ecdsa.PublicKey:
			if k.Curve != elliptic.P256() {
				return errors.New("EC signing keys must use P-256")
			}
			if key.Algorithm != "" && key.Algorithm != "ES256" {
				return errors.New("EC signing keys must use ES256")
			}
		default:
			return errors.New("unsupported public key type")
		}
	}
	return nil
}

// Fosite defaults private_key_jwt to RS256. Require a compatible key before
// accepting the client, including after fetching a referenced key set.
func validateAssertionKeys(m *ClientMetadata) error {
	if m.TokenEndpointAuthMethod != "private_key_jwt" {
		return nil
	}
	if m.TokenEndpointAuthSigningAlg == "" {
		m.TokenEndpointAuthSigningAlg = "RS256"
	}
	if m.TokenEndpointAuthSigningAlg != "RS256" && m.TokenEndpointAuthSigningAlg != "ES256" {
		return errors.New("unsupported client assertion signing algorithm")
	}
	if m.JWKS == nil {
		return nil
	} // The referenced key set is checked after hydration.
	for _, key := range m.JWKS.Keys {
		if key.Algorithm != "" && key.Algorithm != m.TokenEndpointAuthSigningAlg {
			continue
		}
		switch key.Key.(type) {
		case *rsa.PublicKey:
			if m.TokenEndpointAuthSigningAlg == "RS256" {
				return nil
			}
		case *ecdsa.PublicKey:
			if m.TokenEndpointAuthSigningAlg == "ES256" {
				return nil
			}
		}
	}
	return errors.New("no public key supports the selected client assertion algorithm")
}

func metadataURL(raw string, clientID bool) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || strings.ContainsAny(raw, "#\\\r\n") {
		return nil, errors.New("metadata URL must be HTTPS without userinfo or fragment")
	}
	if clientID && (u.Path == "" || u.Path == "/") {
		return nil, errors.New("client_id requires a non-root path")
	}
	for _, segment := range strings.Split(u.Path, "/") {
		if segment == "." || segment == ".." {
			return nil, errors.New("metadata URL contains dot path")
		}
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !publicMetadataIP(ip) {
		return nil, errors.New("metadata URL has a non-public address")
	}
	return u, nil
}

// Restrict IPv6 to ordinary global unicast and exclude IANA special-use ranges.
// In particular, translation prefixes must not bypass IPv4 address checks.
var metadataBlockedPrefixes = func() []netip.Prefix {
	var result []netip.Prefix
	for _, s := range []string{"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.31.196.0/24", "192.52.193.0/24", "192.175.48.0/24", "192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4", "2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20", "2620:4f:8000::/48"} {
		result = append(result, netip.MustParsePrefix(s))
	}
	return result
}()

func publicMetadataIP(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() {
		return false
	}
	if address.Is6() && !netip.MustParsePrefix("2000::/3").Contains(address) {
		return false
	}
	for _, prefix := range metadataBlockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

type metadataCacheEntry struct {
	body    []byte
	expires time.Time
}
type metadataResolver struct {
	http       *http.Client
	mu         sync.Mutex
	cache      map[string]metadataCacheEntry
	fetchSlots chan struct{}
}

func newMetadataHTTPClient(lookup func(context.Context, string) ([]net.IPAddr, error), dial func(context.Context, string, string) (net.Conn, error)) *http.Client {
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true, ForceAttemptHTTP2: false, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := lookup(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, errors.New("metadata host has no addresses")
		}
		// Reject the entire answer when any address is unsafe; never resolve again at dial time.
		for _, ip := range ips {
			if ip.Zone != "" || !publicMetadataIP(ip.IP) {
				return nil, errors.New("metadata host resolves to a non-public address")
			}
		}
		var lastErr error
		for _, ip := range ips {
			conn, err := dial(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
	return &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("metadata redirects are prohibited") }}
}

var defaultMetadataResolver = metadataResolver{http: newMetadataHTTPClient(net.DefaultResolver.LookupIPAddr, (&net.Dialer{Timeout: 5 * time.Second}).DialContext)}

// ResolveMetadata resolves exact client identifiers with a bounded cache.
// Only validated documents and public keys are cached, for at most one hour.
func ResolveMetadata(ctx context.Context, id string) (*ClientMetadata, error) {
	return defaultMetadataResolver.resolve(ctx, id)
}
func (r *metadataResolver) resolve(ctx context.Context, id string) (*ClientMetadata, error) {
	if _, err := metadataURL(id, true); err != nil {
		return nil, err
	}
	r.mu.Lock()
	entry, hit := r.cache[id]
	r.mu.Unlock()
	if hit && time.Now().Before(entry.expires) {
		var cached ClientMetadata
		if err := json.Unmarshal(entry.body, &cached); err == nil {
			return &cached, nil
		}
	}
	started := time.Now()
	var m ClientMetadata
	ttl, err := r.fetch(ctx, id, &m)
	if err != nil {
		return nil, err
	}
	if m.ClientID != id {
		return nil, errors.New("metadata client_id does not exactly match requested URL")
	}
	if err := ValidateMetadata(&m, true); err != nil {
		return nil, err
	}
	keysTTL, err := r.hydrateTTL(ctx, &m)
	if err != nil {
		return nil, err
	}
	ttl = min(ttl, keysTTL) - time.Since(started)
	if ttl > 0 {
		body, err := json.Marshal(&m)
		if err != nil {
			return nil, err
		}
		r.mu.Lock()
		if r.cache == nil {
			r.cache = make(map[string]metadataCacheEntry)
		}
		if len(r.cache) >= 256 {
			for key := range r.cache {
				delete(r.cache, key)
				break
			}
		}
		r.cache[id] = metadataCacheEntry{body: body, expires: time.Now().Add(ttl)}
		r.mu.Unlock()
	}
	return &m, nil
}
func (r *metadataResolver) fetch(ctx context.Context, raw string, dest any) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	r.mu.Lock()
	if r.fetchSlots == nil {
		r.fetchSlots = make(chan struct{}, 16)
	}
	slots := r.fetchSlots
	r.mu.Unlock()
	select {
	case slots <- struct{}{}:
		defer func() { <-slots }()
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	if _, err := metadataURL(raw, false); err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	response, err := r.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("metadata returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil {
		return 0, err
	}
	if len(body) > 65536 {
		return 0, errors.New("metadata exceeds 64 KiB")
	}
	if len(bytes.TrimSpace(body)) == 0 || bytes.TrimSpace(body)[0] != '{' {
		return 0, errors.New("metadata must be one JSON object")
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return 0, err
	}
	return metadataCacheTTL(response.Header), nil
}

// HydrateKeys retrieves and validates public signing keys with the same SSRF
// protections as metadata. Clear JWKSURI afterwards to prevent Fosite fetching it
// through an unrestricted HTTP client.
func HydrateKeys(ctx context.Context, m *ClientMetadata) error {
	return defaultMetadataResolver.hydrate(ctx, m)
}
func (r *metadataResolver) hydrate(ctx context.Context, m *ClientMetadata) error {
	_, err := r.hydrateTTL(ctx, m)
	return err
}
func (r *metadataResolver) hydrateTTL(ctx context.Context, m *ClientMetadata) (time.Duration, error) {
	ttl := time.Hour
	if m.JWKSURI != "" {
		if m.JWKS != nil {
			return 0, errors.New("jwks and jwks_uri are mutually exclusive")
		}
		var keys jose.JSONWebKeySet
		var err error
		ttl, err = r.fetch(ctx, m.JWKSURI, &keys)
		if err != nil {
			return 0, err
		}
		if err := validatePublicKeys(&keys); err != nil {
			return 0, err
		}
		m.JWKS = &keys
		m.JWKSURI = ""
	}
	if err := validatePublicKeys(m.JWKS); err != nil {
		return 0, err
	}
	return ttl, validateAssertionKeys(m)
}

func metadataCacheTTL(header http.Header) time.Duration {
	ttl := 5 * time.Minute
	for _, line := range header.Values("Cache-Control") {
		for _, part := range strings.Split(line, ",") {
			name, value, _ := strings.Cut(strings.TrimSpace(part), "=")
			switch strings.ToLower(name) {
			case "no-store", "no-cache":
				return 0
			case "max-age":
				seconds, err := strconv.ParseInt(strings.Trim(value, "\""), 10, 64)
				if err != nil || seconds <= 0 {
					return 0
				}
				if seconds <= 3600 {
					ttl = time.Duration(seconds) * time.Second
				} else {
					ttl = time.Hour
				}
			}
		}
	}
	if age, err := strconv.ParseInt(header.Get("Age"), 10, 64); err == nil && age > 0 {
		if age >= int64(ttl/time.Second) {
			return 0
		}
		ttl -= time.Duration(age) * time.Second
	}
	return ttl
}

func MetadataClient(m *ClientMetadata) *fosite.DefaultOpenIDConnectClient {
	return &fosite.DefaultOpenIDConnectClient{
		DefaultClient: &fosite.DefaultClient{
			ID:            m.ClientID,
			RedirectURIs:  slices.Clone(m.RedirectURIs),
			GrantTypes:    slices.Clone(m.GrantTypes),
			ResponseTypes: slices.Clone(m.ResponseTypes),
			Scopes:        strings.Fields(m.Scope),
			Public:        m.TokenEndpointAuthMethod == "none",
		},
		JSONWebKeys:                       m.JWKS,
		TokenEndpointAuthMethod:           m.TokenEndpointAuthMethod,
		TokenEndpointAuthSigningAlgorithm: m.TokenEndpointAuthSigningAlg,
	}
}
