package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ory/fosite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"within.website/x/cmd/iamd/models"
)

var baseScopes = []string{"openid", "profile", "email", "offline_access"}

func (s *store) catalogScopes(ctx context.Context) []string {
	out := slices.Clone(baseScopes)
	var resources []resource
	if err := s.dbFor(ctx).Find(&resources).Error; err != nil {
		return out
	}
	for _, r := range resources {
		for _, scope := range strings.Fields(r.Scopes) {
			if !slices.Contains(out, scope) {
				out = append(out, scope)
			}
		}
	}
	return out
}
func (s *store) clientMetadata(ctx context.Context, id string) (*ClientMetadata, []byte, string, error) {
	var rec clientRecord
	err := s.dbFor(ctx).Where("id = ?", id).First(&rec).Error
	if err == nil && rec.Disabled {
		return nil, nil, "", fosite.ErrNotFound
	}
	var m *ClientMetadata
	var binding string
	if strings.HasPrefix(id, "https://") {
		// URL metadata is fetched afresh at authorization and token authentication.
		// A local row only records administrative disable state.
		m, err = ResolveMetadata(ctx, id)
		if err != nil {
			metadataFailures.Inc()
			return nil, nil, "", fosite.ErrNotFound
		}
		encoded, e := json.Marshal(m)
		if e != nil {
			return nil, nil, "", e
		}
		binding = hash(string(encoded))
		if err := s.dbFor(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&clientRecord{ID: id, Metadata: encoded}).Error; err != nil {
			return nil, nil, "", err
		}
	} else {
		if err != nil {
			return nil, nil, "", fosite.ErrNotFound
		}
		m = &ClientMetadata{}
		if err := json.Unmarshal(rec.Metadata, m); err != nil {
			return nil, nil, "", err
		}
		binding = hash(string(rec.Metadata))
		if err := ValidateMetadata(m, false); err != nil {
			return nil, nil, "", err
		}
	}
	if m.Scope == "" {
		m.Scope = "openid profile email"
	}
	allowed := s.catalogScopes(ctx)
	for _, scope := range strings.Fields(m.Scope) {
		if !slices.Contains(allowed, scope) {
			return nil, nil, "", fosite.ErrInvalidScope
		}
	}
	return m, rec.SecretHash, binding, nil
}
func (s *store) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	m, secret, _, err := s.clientMetadata(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := HydrateKeys(ctx, m); err != nil {
		return nil, fosite.ErrNotFound
	}
	client := MetadataClient(m)
	client.Secret = secret
	var resources []resource
	if err := s.dbFor(ctx).Find(&resources).Error; err != nil {
		return nil, err
	}
	for _, r := range resources {
		client.Audience = append(client.Audience, r.Audience)
	}
	client.Audience = append(client.Audience, s.issuer+"/userinfo")
	return client, nil
}
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	// SQLite counters make limits survive restarts. Restrict both aggregate and
	// source rates because registration is deliberately unauthenticated.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !s.allowRate(r.Context(), "register:all", 100) || !s.allowRate(r.Context(), "register:"+host, 10) {
		registrationProblem(w, "slow_down", 429)
		return
	}
	var m ClientMetadata
	if err := readJSON(r, &m); err != nil {
		registrationProblem(w, "invalid_client_metadata", 400)
		return
	}
	if m.ClientID != "" {
		registrationProblem(w, "invalid_client_metadata", 400)
		return
	}
	m.ClientID = randomToken()
	if err := ValidateMetadata(&m, false); err != nil {
		registrationProblem(w, "invalid_client_metadata", 400)
		return
	}
	if m.Scope == "" {
		m.Scope = "openid profile email"
	}
	for _, scope := range strings.Fields(m.Scope) {
		if !slices.Contains(s.store.catalogScopes(r.Context()), scope) {
			registrationProblem(w, "invalid_client_metadata", 400)
			return
		}
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		registrationProblem(w, "server_error", 500)
		return
	}
	// Validate externally supplied keys using the same pinned resolver as CIMD.
	check := m
	if err := HydrateKeys(r.Context(), &check); err != nil {
		registrationProblem(w, "invalid_client_metadata", 400)
		return
	}
	rec := clientRecord{ID: m.ClientID, Metadata: encoded}
	var secret string
	if m.TokenEndpointAuthMethod == "client_secret_basic" {
		secret = randomToken()
		rec.SecretHash, err = bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			registrationProblem(w, "server_error", 500)
			return
		}
	}
	if err := s.store.db.WithContext(r.Context()).Create(&rec).Error; err != nil {
		registrationProblem(w, "server_error", 500)
		return
	}
	var doc map[string]any
	_ = json.Unmarshal(encoded, &doc)
	doc["client_id_issued_at"] = time.Now().Unix()
	if secret != "" {
		doc["client_secret"] = secret
		doc["client_secret_expires_at"] = 0
	}
	writeJSON(w, 201, doc)
}
func (s *Server) allowRate(ctx context.Context, key string, limit int) bool {
	allowed := false
	err := s.store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix() / 60
		var row rateRecord
		err := tx.Where("key = ?", hash(key)).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = rateRecord{Key: hash(key), Window: now, Count: 1}
			allowed = true
			return tx.Create(&row).Error
		}
		if err != nil {
			return err
		}
		if row.Window != now {
			row.Window = now
			row.Count = 0
		}
		row.Count++
		allowed = row.Count <= limit
		return tx.Save(&row).Error
	})
	return err == nil && allowed
}
func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		problem(w, "invalid_request", 400)
		return
	}
	// Never send errors to a redirect until every client and protocol input is
	// checked. Browser login and consent must not hide malformed requests.
	for _, v := range r.Form {
		if len(v) != 1 {
			problem(w, "invalid_request", 400)
			return
		}
	}
	if r.Form.Get("response_type") != "code" || r.Form.Get("response_mode") != "" && r.Form.Get("response_mode") != "query" || r.Form.Get("request") != "" || r.Form.Get("request_uri") != "" {
		problem(w, "unsupported_response_type", 400)
		return
	}
	m, _, binding, err := s.store.clientMetadata(r.Context(), r.Form.Get("client_id"))
	if err != nil {
		problem(w, "invalid_client", 400)
		return
	}
	if !slices.Contains(m.GrantTypes, "authorization_code") || !slices.Contains(m.ResponseTypes, "code") {
		problem(w, "unauthorized_client", 400)
		return
	}
	if !validRedirect(m, r.Form.Get("redirect_uri")) {
		problem(w, "invalid_request", 400)
		return
	}
	challenge := r.Form.Get("code_challenge")
	if len(challenge) != 43 || r.Form.Get("code_challenge_method") != "S256" || !validBase64URL(challenge) {
		problem(w, "invalid_request", 400)
		return
	}
	if nonce := r.Form.Get("nonce"); nonce != "" && len(nonce) < 8 {
		problem(w, "invalid_request", 400)
		return
	}
	if state := r.Form.Get("state"); state != "" && len(state) < 8 {
		problem(w, "invalid_request", 400)
		return
	}
	prompts := strings.Fields(r.Form.Get("prompt"))
	for _, p := range prompts {
		if !slices.Contains([]string{"none", "login", "consent", "select_account"}, p) {
			problem(w, "invalid_request", 400)
			return
		}
	}
	if slices.Contains(prompts, "none") && len(prompts) > 1 {
		problem(w, "invalid_request", 400)
		return
	}
	var maxAge *int64
	if raw, ok := r.Form["max_age"]; ok {
		value, err := strconv.ParseInt(raw[0], 10, 64)
		if err != nil || value < 0 || value > 315360000 {
			problem(w, "invalid_request", 400)
			return
		}
		maxAge = &value
	}
	scopes := strings.Fields(r.Form.Get("scope"))
	if !slices.Contains(scopes, "openid") {
		problem(w, "invalid_scope", 400)
		return
	}
	requestedAudience := r.Form.Get("resource")
	if requestedAudience == "" {
		requestedAudience = s.cfg.Issuer + "/userinfo"
	}
	allowedScopes := slices.Clone(baseScopes)
	if requestedAudience != s.cfg.Issuer+"/userinfo" {
		var res resource
		if err := s.store.db.WithContext(r.Context()).Where("audience = ?", requestedAudience).First(&res).Error; err != nil {
			problem(w, "invalid_target", 400)
			return
		}
		allowedScopes = append(allowedScopes, strings.Fields(res.Scopes)...)
	}
	for _, scope := range scopes {
		if !slices.Contains(strings.Fields(m.Scope), scope) || !slices.Contains(allowedScopes, scope) {
			problem(w, "invalid_scope", 400)
			return
		}
	}
	if slices.Contains(scopes, "offline_access") && !slices.Contains(m.GrantTypes, "refresh_token") {
		problem(w, "invalid_scope", 400)
		return
	}
	// Fosite validates the complete authorization request before interaction.
	ar, err := s.provider.NewAuthorizeRequest(r.Context(), r)
	if err != nil {
		problem(w, fosite.ErrorToRFC6749Error(err).ErrorField, 400)
		return
	}
	if !ar.GetResponseTypes().ExactOne("code") {
		problem(w, "unsupported_response_type", 400)
		return
	}
	ar = &issuerAuthorizeRequest{AuthorizeRequester: ar, issuer: s.cfg.Issuer}
	browser, user, err := s.browser(r)
	mustLogin := err != nil
	if !mustLogin && (slices.Contains(prompts, "login") || slices.Contains(prompts, "select_account")) {
		mustLogin = true
	}
	if !mustLogin && maxAge != nil && time.Since(browser.AuthTime) > time.Duration(*maxAge)*time.Second {
		mustLogin = true
	}
	// The login continuation is bound to this exact authorization request and
	// stays available while the browser displays and submits consent. Consume
	// it only when issuing the authorization response.
	var continuation *pendingLogin
	if r.Form.Get("transaction") != "" && err == nil {
		q := cloneValues(r.Form)
		q.Del("transaction")
		q.Del("csrf")
		q.Del("decision")
		var completed pendingLogin
		if e := s.store.db.WithContext(r.Context()).Where("hash = ? AND browser_hash = ? AND expires_at > ?", hash(r.Form.Get("transaction")), browser.Hash, time.Now()).First(&completed).Error; e == nil && completed.ReturnTo == "/authorize?"+q.Encode() {
			continuation = &completed
			mustLogin = false
		}
	}
	if mustLogin {
		if slices.Contains(prompts, "none") {
			s.provider.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrLoginRequired)
			return
		}
		q := cloneValues(r.Form)
		q.Del("csrf")
		q.Del("decision")
		q.Del("transaction")
		s.beginLogin(w, r, "/authorize?"+q.Encode())
		return
	}
	var grant consent
	grantErr := s.store.db.WithContext(r.Context()).Where("user_uuid = ? AND client_id = ?", user.UUID, m.ClientID).First(&grant).Error
	hasConsent := grantErr == nil && !grant.Revoked && grant.Audience == requestedAudience && grant.MetadataHash == binding
	for _, scope := range scopes {
		hasConsent = hasConsent && slices.Contains(strings.Fields(grant.Scopes), scope)
	}
	if slices.Contains(prompts, "consent") {
		hasConsent = false
	}
	if r.Method == "POST" {
		if !s.validCSRF(r, browser) {
			problem(w, "invalid_request", 403)
			return
		}
		if r.Form.Get("decision") != "approve" {
			s.provider.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrAccessDenied)
			return
		}
		grant = consent{UserUUID: user.UUID, ClientID: m.ClientID, Scopes: strings.Join(scopes, " "), Audience: requestedAudience, MetadataHash: binding, CreatedAt: time.Now()}
		if err := s.store.db.WithContext(r.Context()).Save(&grant).Error; err != nil {
			problem(w, "server_error", 500)
			return
		}
		hasConsent = true
	}
	if !hasConsent {
		if slices.Contains(prompts, "none") {
			s.provider.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrConsentRequired)
			return
		}
		s.renderConsent(w, r, m, scopes, requestedAudience, browser.CSRF, r.Form)
		return
	}
	for _, scope := range scopes {
		ar.GrantScope(scope)
	}
	ar.GrantAudience(requestedAudience)
	sess := newSession(user.UUID, browser.Hash, browser.AuthTime)
	sess.MetadataHash = binding
	sess.AccessClaims.Extra = map[string]any{"client_id": m.ClientID}
	sess.Claims.RequestedAt = browser.AuthTime
	s.addProfileClaims(r.Context(), sess, user.UUID, scopes)
	if continuation != nil {
		result := s.store.db.WithContext(r.Context()).Where("hash = ?", continuation.Hash).Delete(&pendingLogin{})
		if result.Error != nil || result.RowsAffected != 1 {
			problem(w, "invalid_request", 400)
			return
		}
	}
	resp, err := s.provider.NewAuthorizeResponse(r.Context(), ar, sess)
	if err != nil {
		s.provider.WriteAuthorizeError(r.Context(), w, ar, err)
		return
	}
	s.provider.WriteAuthorizeResponse(r.Context(), w, ar, resp)
}
func validBase64URL(s string) bool {
	for _, c := range s {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func validRedirect(m *ClientMetadata, raw string) bool {
	if slices.Contains(m.RedirectURIs, raw) {
		return true
	}
	if m.ApplicationType != "native" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Fragment != "" {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return false
	}
	for _, registered := range m.RedirectURIs {
		v, err := url.Parse(registered)
		if err == nil && u.Scheme == v.Scheme && u.Hostname() == v.Hostname() && u.EscapedPath() == v.EscapedPath() && u.RawQuery == v.RawQuery && u.ForceQuery == v.ForceQuery {
			return true
		}
	}
	return false
}
func cloneValues(in url.Values) url.Values {
	out := url.Values{}
	for k, v := range in {
		out[k] = slices.Clone(v)
	}
	return out
}
func (s *Server) token(w http.ResponseWriter, r *http.Request) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	if err := r.ParseForm(); err != nil {
		problem(w, "invalid_request", 400)
		return
	}
	for _, v := range r.PostForm {
		if len(v) != 1 {
			problem(w, "invalid_request", 400)
			return
		}
	}
	if r.PostForm.Get("grant_type") != "authorization_code" && r.PostForm.Get("grant_type") != "refresh_token" {
		problem(w, "unsupported_grant_type", 400)
		return
	}
	if _, err := s.signingKey(r.Context()); err != nil {
		problem(w, "server_error", 500)
		return
	}
	ar, err := s.provider.NewAccessRequest(r.Context(), r, newSession("", "", time.Time{}))
	if err != nil {
		if errors.Is(err, fosite.ErrJTIKnown) {
			err = fosite.ErrInvalidClient.WithWrap(err)
		}
		s.provider.WriteAccessError(r.Context(), w, ar, err)
		return
	}
	// Fosite copies code-granted scopes and audiences during response
	// population. Hydrate them first so our persisted consent is checked before
	// any code is consumed or token is written.
	if r.PostForm.Get("grant_type") == "authorization_code" {
		var record oauthRecord
		if err := s.store.db.WithContext(r.Context()).Where("kind = ? AND request_id = ? AND active = ?", "code", ar.GetID(), true).First(&record).Error; err != nil {
			problem(w, "invalid_grant", 400)
			return
		}
		original, err := decodeRequest(record.Data)
		if err != nil {
			problem(w, "invalid_grant", 400)
			return
		}
		for _, scope := range original.GetGrantedScopes() {
			ar.GrantScope(scope)
		}
		for _, audience := range original.GetGrantedAudience() {
			ar.GrantAudience(audience)
		}
	}
	sess, ok := ar.GetSession().(*session)
	if !ok {
		problem(w, "invalid_grant", 400)
		return
	}
	_, _, binding, err := s.store.clientMetadata(r.Context(), ar.GetClient().GetID())
	if err != nil || binding != sess.MetadataHash {
		problem(w, "invalid_grant", 400)
		return
	}
	var user models.User
	if err := s.store.db.WithContext(r.Context()).Where("uuid = ?", sess.Subject).First(&user).Error; err != nil {
		problem(w, "invalid_grant", 400)
		return
	}
	var grant consent
	if err := s.store.db.WithContext(r.Context()).Where("user_uuid = ? AND client_id = ? AND revoked = ?", sess.Subject, ar.GetClient().GetID(), false).First(&grant).Error; err != nil {
		problem(w, "invalid_grant", 400)
		return
	}
	if grant.MetadataHash != binding || len(ar.GetGrantedAudience()) != 1 || grant.Audience != ar.GetGrantedAudience()[0] {
		problem(w, "invalid_grant", 400)
		return
	}
	allowedScopes := slices.Clone(baseScopes)
	if grant.Audience != s.cfg.Issuer+"/userinfo" {
		var current resource
		if err := s.store.db.WithContext(r.Context()).Where("audience = ?", grant.Audience).First(&current).Error; err != nil {
			problem(w, "invalid_grant", 400)
			return
		}
		allowedScopes = append(allowedScopes, strings.Fields(current.Scopes)...)
	}
	for _, scope := range ar.GetGrantedScopes() {
		if !slices.Contains(strings.Fields(grant.Scopes), scope) || !slices.Contains(allowedScopes, scope) {
			problem(w, "invalid_grant", 400)
			return
		}
	}
	sess.AccessClaims.IssuedAt = time.Time{}
	sess.AccessClaims.JTI = ""
	sess.AccessClaims.Extra = map[string]any{"client_id": ar.GetClient().GetID()}
	response, err := s.provider.NewAccessResponse(r.Context(), ar)
	if err != nil {
		s.provider.WriteAccessError(r.Context(), w, ar, err)
		return
	}
	s.provider.WriteAccessResponse(r.Context(), w, ar, response)
}
func (s *Server) revoke(w http.ResponseWriter, r *http.Request) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	err := s.provider.NewRevocationRequest(r.Context(), r)
	s.provider.WriteRevocationResponse(r.Context(), w, err)
}
func (s *Server) userinfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		problem(w, "invalid_token", 401)
		return
	}
	claims, err := s.verifyJWT(r.Context(), strings.TrimPrefix(auth, "Bearer "), "at+JWT", s.cfg.Issuer+"/userinfo")
	if err != nil {
		problem(w, "invalid_token", 401)
		return
	}
	parts := strings.Split(strings.TrimPrefix(auth, "Bearer "), ".")
	if len(parts) != 3 {
		problem(w, "invalid_token", 401)
		return
	}
	active, err := s.store.GetAccessTokenSession(r.Context(), parts[2], nil)
	if err != nil {
		problem(w, "invalid_token", 401)
		return
	}
	var registered clientRecord
	clientID := active.GetClient().GetID()
	registryErr := s.store.db.WithContext(r.Context()).Where("id = ?", clientID).First(&registered).Error
	// URL clients authenticate from their metadata document and need no DCR
	// row. An explicit administrative disable record still takes precedence.
	missingURLClient := errors.Is(registryErr, gorm.ErrRecordNotFound) && strings.HasPrefix(clientID, "https://")
	if registryErr != nil && !missingURLClient || registryErr == nil && registered.Disabled {
		problem(w, "invalid_token", 401)
		return
	}
	var grant consent
	if err := s.store.db.WithContext(r.Context()).Where("user_uuid = ? AND client_id = ? AND revoked = ?", active.GetSession().GetSubject(), active.GetClient().GetID(), false).First(&grant).Error; err != nil {
		problem(w, "invalid_token", 401)
		return
	}
	sub, _ := claims["sub"].(string)
	user, err := s.store.dao.GetUser(r.Context(), sub)
	if err != nil {
		problem(w, "invalid_token", 401)
		return
	}
	scope, _ := claims["scope"].(string)
	scopes := strings.Fields(scope)
	if !slices.Contains(scopes, "openid") {
		w.Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope"`)
		problem(w, "insufficient_scope", 403)
		return
	}
	result := map[string]any{"sub": user.UUID}
	var ident identity
	if slices.Contains(scopes, "profile") {
		result["name"] = user.Name
	}
	if slices.Contains(scopes, "email") && s.store.db.WithContext(r.Context()).Where("user_uuid = ? AND issuer = ?", user.UUID, s.cfg.UpstreamIssuer).First(&ident).Error == nil {
		result["email"] = ident.Email
		result["email_verified"] = ident.EmailVerified
	}
	w.Header().Del("WWW-Authenticate")
	writeJSON(w, 200, result)
}
func (s *Server) addProfileClaims(ctx context.Context, sess *session, sub string, scopes []string) {
	var ident identity
	if s.store.db.WithContext(ctx).Where("user_uuid = ? AND issuer = ?", sub, s.cfg.UpstreamIssuer).First(&ident).Error != nil {
		return
	}
	extra := map[string]any{}
	if slices.Contains(scopes, "profile") {
		extra["name"] = ident.Name
	}
	if slices.Contains(scopes, "email") {
		extra["email"] = ident.Email
		extra["email_verified"] = ident.EmailVerified
	}
	sess.Claims.Extra = extra
}

// Preserve Fosite's validation against the unmodified registered redirect while
// adding RFC 9207 issuer identification to the emitted response URL.
type issuerAuthorizeRequest struct {
	fosite.AuthorizeRequester
	issuer string
}

func (r *issuerAuthorizeRequest) GetRedirectURI() *url.URL {
	original := r.AuthorizeRequester.GetRedirectURI()
	if original == nil {
		return nil
	}
	copy := *original
	q := copy.Query()
	q.Set("iss", r.issuer)
	copy.RawQuery = q.Encode()
	return &copy
}
