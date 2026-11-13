// Package oidc implements iamd's durable OpenID Connect provider. Its HTTP
// adapter is called by the typed Connect transport; it is not mounted directly.
package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	upstream "github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v3"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	fjwt "github.com/ory/fosite/token/jwt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"within.website/x/cmd/iamd/models"
)

type Config struct{ Issuer, UpstreamIssuer, UpstreamClientID, UpstreamClientSecret string }
type Server struct {
	cfg              Config
	store            *store
	provider         fosite.OAuth2Provider
	upstreamProvider *upstream.Provider
	upstreamOAuth    oauth2.Config
	keyMu            sync.RWMutex
	currentKey       keyRecord
	privateKey       *rsa.PrivateKey
	// A single replica serializes token transactions, including Fosite's reads
	// before its response transaction. Replayed refresh tokens revoke the family.
	tokenMu sync.Mutex
}

func New(ctx context.Context, dao *models.DAO, cfg Config) (*Server, error) {
	st, err := newStore(dao)
	if err != nil {
		return nil, err
	}
	return newServer(ctx, st, cfg, true)
}
func newServer(ctx context.Context, st *store, cfg Config, discoverUpstream bool) (*Server, error) {
	cfg.Issuer = strings.TrimRight(cfg.Issuer, "/")
	iss, err := url.Parse(cfg.Issuer)
	if err != nil || iss.Scheme != "https" || iss.Host == "" || iss.RawQuery != "" || iss.Fragment != "" || iss.User != nil {
		return nil, errors.New("OIDC issuer must be an HTTPS URL without query, fragment or credentials")
	}
	if cfg.UpstreamIssuer == "" {
		cfg.UpstreamIssuer = "https://idp.xeserv.us"
	}
	if cfg.UpstreamIssuer != "https://idp.xeserv.us" {
		return nil, errors.New("OIDC upstream must be https://idp.xeserv.us")
	}
	st.issuer = cfg.Issuer
	s := &Server{cfg: cfg, store: st}
	if err := s.loadKeys(ctx); err != nil {
		return nil, err
	}
	var secret secretRecord
	if err := st.db.Where("name = ?", "hmac").First(&secret).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		secret = secretRecord{Name: "hmac", Value: []byte(randomToken())}
		if err := st.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&secret).Error; err != nil {
			return nil, err
		}
		if err := st.db.Where("name = ?", "hmac").First(&secret).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	config := &fosite.Config{GlobalSecret: secret.Value, AccessTokenLifespan: 10 * time.Minute, IDTokenLifespan: 10 * time.Minute, AuthorizeCodeLifespan: 5 * time.Minute, RefreshTokenLifespan: 30 * 24 * time.Hour, EnforcePKCE: true, EnablePKCEPlainChallengeMethod: false, TokenURL: cfg.Issuer + "/token", AccessTokenIssuer: cfg.Issuer, IDTokenIssuer: cfg.Issuer, RefreshTokenScopes: []string{"offline_access"}, JWTScopeClaimKey: fjwt.JWTScopeFieldString, AllowedPromptValues: []string{"none", "login", "consent", "select_account"}, SanitationWhiteList: []string{"nonce", "redirect_uri", "code_challenge", "code_challenge_method", "max_age", "prompt", "resource"}}
	signer := &tokenSigner{server: s, DefaultSigner: &fjwt.DefaultSigner{GetPrivateKey: s.signingKey}}
	core := compose.NewOAuth2JWTStrategy(s.signingKey, compose.NewOAuth2HMACStrategy(config), config)
	core.Signer = signer
	oidcStrategy := compose.NewOpenIDConnectStrategy(s.signingKey, config)
	oidcStrategy.Signer = signer
	s.provider = compose.Compose(config, st, &compose.CommonStrategy{CoreStrategy: core, OpenIDConnectTokenStrategy: oidcStrategy, Signer: signer}, compose.OAuth2AuthorizeExplicitFactory, compose.OAuth2RefreshTokenGrantFactory, compose.OpenIDConnectExplicitFactory, compose.OpenIDConnectRefreshFactory, compose.OAuth2TokenRevocationFactory, compose.OAuth2PKCEFactory)
	if discoverUpstream {
		if cfg.UpstreamClientID == "" || cfg.UpstreamClientSecret == "" {
			return nil, errors.New("upstream OIDC client ID and secret are required")
		}
		p, err := upstream.NewProvider(ctx, cfg.UpstreamIssuer)
		if err != nil {
			return nil, fmt.Errorf("discover upstream OIDC: %w", err)
		}
		s.upstreamProvider = p
		s.upstreamOAuth = oauth2.Config{ClientID: cfg.UpstreamClientID, ClientSecret: cfg.UpstreamClientSecret, Endpoint: p.Endpoint(), RedirectURL: cfg.Issuer + "/upstream/callback", Scopes: []string{"openid", "profile", "email"}}
	}
	return s, nil
}
func (s *Server) loadKeys(ctx context.Context) error {
	var key keyRecord
	err := s.store.db.WithContext(ctx).Where("retired_at IS NULL").Order("created_at DESC").First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.rotateKey(ctx)
	}
	if err != nil {
		return err
	}
	private, err := x509.ParsePKCS1PrivateKey(key.DER)
	if err != nil {
		return fmt.Errorf("read OIDC signing key: %w", err)
	}
	s.currentKey = key
	s.privateKey = private
	if time.Since(key.CreatedAt) >= 30*24*time.Hour {
		return s.rotateKey(ctx)
	}
	return nil
}
func (s *Server) rotateKey(ctx context.Context) error {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	key := keyRecord{ID: randomToken(), DER: x509.MarshalPKCS1PrivateKey(private), CreatedAt: now}
	err = s.store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&keyRecord{}).Where("retired_at IS NULL").Update("retired_at", now).Error; err != nil {
			return err
		}
		return tx.Create(&key).Error
	})
	if err != nil {
		return err
	}
	s.currentKey = key
	s.privateKey = private
	return nil
}
func (s *Server) signingKey(ctx context.Context) (interface{}, error) {
	s.keyMu.RLock()
	expired := time.Since(s.currentKey.CreatedAt) >= 30*24*time.Hour
	s.keyMu.RUnlock()
	if expired {
		if ctx.Value(txKey{}) != nil {
			return nil, errors.New("signing key rotation required before SQL transaction")
		}
		if err := s.rotateKey(ctx); err != nil {
			return nil, err
		}
	}
	s.keyMu.RLock()
	defer s.keyMu.RUnlock()
	return &jose.JSONWebKey{Key: s.privateKey, KeyID: s.currentKey.ID, Algorithm: "RS256", Use: "sig"}, nil
}
func (s *Server) publicKeys(ctx context.Context) (jose.JSONWebKeySet, error) {
	var rows []keyRecord
	if err := s.store.db.WithContext(ctx).Where("retired_at IS NULL OR retired_at > ?", time.Now().Add(-11*time.Minute)).Find(&rows).Error; err != nil {
		return jose.JSONWebKeySet{}, err
	}
	keys := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{}}
	for _, row := range rows {
		key, err := x509.ParsePKCS1PrivateKey(row.DER)
		if err != nil {
			return keys, err
		}
		keys.Keys = append(keys.Keys, jose.JSONWebKey{Key: &key.PublicKey, KeyID: row.ID, Algorithm: "RS256", Use: "sig"})
	}
	return keys, nil
}

// Fosite's default signer fixes typ to JWT. This signer supplies the RFC 9068
// at+JWT access-token type while leaving ID tokens explicitly typed JWT.
type tokenSigner struct {
	*fjwt.DefaultSigner
	server *Server
}

func (s *tokenSigner) Generate(ctx context.Context, claims fjwt.MapClaims, header fjwt.Mapper) (string, string, error) {
	key, err := s.server.signingKey(ctx)
	if err != nil {
		return "", "", err
	}
	jwk := key.(*jose.JSONWebKey)
	opts := (&jose.SignerOptions{}).WithType("JWT")
	for k, v := range header.ToMap() {
		opts.WithHeader(jose.HeaderKey(k), v)
	}
	opts.WithHeader("kid", jwk.KeyID)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk.Key}, opts)
	if err != nil {
		return "", "", err
	}
	b, err := json.Marshal(claims)
	if err != nil {
		return "", "", err
	}
	signed, err := signer.Sign(b)
	if err != nil {
		return "", "", err
	}
	raw, err := signed.CompactSerialize()
	if err != nil {
		return "", "", err
	}
	return raw, strings.Split(raw, ".")[2], nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	path := r.URL.Path
	switch {
	case path == "/.well-known/openid-configuration" || path == "/.well-known/oauth-authorization-server":
		if !method(w, r, "GET") {
			return
		}
		s.discovery(w, r)
	case path == "/jwks.json":
		if !method(w, r, "GET") {
			return
		}
		keys, err := s.publicKeys(r.Context())
		if err != nil {
			problem(w, "server_error", 500)
			return
		}
		writeJSON(w, 200, keys)
	case path == "/authorize":
		if method(w, r, "GET", "POST") {
			s.authorize(w, r)
		}
	case path == "/token":
		if method(w, r, "POST") {
			s.token(w, r)
		}
	case path == "/userinfo":
		if method(w, r, "GET", "POST") {
			s.userinfo(w, r)
		}
	case path == "/register":
		if method(w, r, "POST") {
			s.register(w, r)
		}
	case path == "/revoke":
		if method(w, r, "POST") {
			s.revoke(w, r)
		}
	case path == "/upstream/callback":
		if method(w, r, "GET") {
			s.callback(w, r)
		}
	case path == "/upstream/backchannel-logout":
		if method(w, r, "POST") {
			s.backchannelLogout(w, r)
		}
	case path == "/login":
		if method(w, r, "GET") {
			s.beginLogin(w, r, "/account")
		}
	case path == "/account":
		if method(w, r, "GET", "POST") {
			s.account(w, r)
		}
	case path == "/consent":
		if method(w, r, "GET", "POST") {
			s.authorize(w, r)
		}
	case path == "/logout":
		if method(w, r, "GET", "POST") {
			s.logout(w, r)
		}
	case strings.HasPrefix(path, "/admin/"):
		s.admin(w, r)
	default:
		http.NotFound(w, r)
	}
}
func method(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	for _, v := range allowed {
		if r.Method == v {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	problem(w, "invalid_request", 405)
	return false
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, code string, status int) {
	writeJSON(w, status, map[string]string{"error": code})
}
func (s *Server) discovery(w http.ResponseWriter, r *http.Request) {
	i := s.cfg.Issuer
	writeJSON(w, 200, map[string]any{"issuer": i, "authorization_endpoint": i + "/authorize", "token_endpoint": i + "/token", "userinfo_endpoint": i + "/userinfo", "jwks_uri": i + "/jwks.json", "registration_endpoint": i + "/register", "revocation_endpoint": i + "/revoke", "end_session_endpoint": i + "/logout", "response_types_supported": []string{"code"}, "response_modes_supported": []string{"query"}, "grant_types_supported": []string{"authorization_code", "refresh_token"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}, "token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "private_key_jwt"}, "token_endpoint_auth_signing_alg_values_supported": []string{"RS256", "ES256"}, "code_challenge_methods_supported": []string{"S256"}, "scopes_supported": s.store.catalogScopes(r.Context()), "claims_supported": []string{"sub", "name", "email", "email_verified", "auth_time"}, "authorization_response_iss_parameter_supported": true, "client_id_metadata_document_supported": true})
}
func (s *Server) verifyJWT(ctx context.Context, raw, typ, aud string) (map[string]any, error) {
	token, err := josejwt.ParseSigned(raw)
	if err != nil {
		return nil, err
	}
	if len(token.Headers) != 1 || token.Headers[0].Algorithm != "RS256" || token.Headers[0].ExtraHeaders["typ"] != typ {
		return nil, errors.New("invalid token type")
	}
	keys, err := s.publicKeys(ctx)
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	for _, key := range keys.Keys {
		if key.KeyID != token.Headers[0].KeyID {
			continue
		}
		if err := token.Claims(key.Key, &claims); err != nil {
			return nil, err
		}
		var registered josejwt.Claims
		if err := token.Claims(key.Key, &registered); err != nil {
			return nil, err
		}
		if err := registered.Validate(josejwt.Expected{Issuer: s.cfg.Issuer, Audience: josejwt.Audience{aud}, Time: time.Now()}); err != nil {
			return nil, err
		}
		if registered.Expiry == nil || registered.Subject == "" {
			return nil, errors.New("missing claims")
		}
		return claims, nil
	}
	return nil, errors.New("unknown key")
}
func readJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
