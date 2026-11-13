package oidc

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	upstream "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"within.website/x/cmd/iamd/models"
)

const sessionCookie = "__Host-iamd_session"
const loginCookie = "__Host-iamd_login"

func setCookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
func (s *Server) browser(r *http.Request) (*browserSession, *models.User, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, nil, err
	}
	var browser browserSession
	if err := s.store.db.WithContext(r.Context()).Where("hash = ? AND revoked = ? AND expires_at > ?", hash(cookie.Value), false, time.Now()).First(&browser).Error; err != nil {
		return nil, nil, err
	}
	user, err := s.store.dao.GetUser(r.Context(), browser.UserUUID)
	if err != nil {
		return nil, nil, err
	}
	return &browser, user, nil
}
func (s *Server) validCSRF(r *http.Request, b *browserSession) bool {
	got := r.Form.Get("csrf")
	if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(b.CSRF)) != 1 {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme+"://"+u.Host != issuerOrigin(s.cfg.Issuer) {
			return false
		}
	}
	return true
}
func (s *Server) beginLogin(w http.ResponseWriter, r *http.Request, returnTo string) {
	if s.upstreamProvider == nil {
		problem(w, "temporarily_unavailable", 503)
		return
	}
	if returnTo != "/account" && !strings.HasPrefix(returnTo, "/authorize?") {
		problem(w, "invalid_request", 400)
		return
	}
	state, nonce, verifier, binding := randomToken(), randomToken(), randomToken(), randomToken()
	pending := pendingLogin{Hash: hash(state), BrowserHash: hash(binding), Nonce: nonce, Verifier: verifier, ReturnTo: returnTo, ExpiresAt: time.Now().Add(5 * time.Minute)}
	if err := s.store.db.WithContext(r.Context()).Create(&pending).Error; err != nil {
		problem(w, "server_error", 500)
		return
	}
	setCookie(w, loginCookie, binding, 300)
	auth := s.upstreamOAuth.AuthCodeURL(state, upstream.Nonce(nonce), oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("prompt", "login"))
	http.Redirect(w, r, auth, http.StatusFound)
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	if s.upstreamProvider == nil {
		problem(w, "temporarily_unavailable", 503)
		return
	}
	if err := r.ParseForm(); err != nil {
		problem(w, "invalid_request", 400)
		return
	}
	for _, v := range r.Form {
		if len(v) != 1 {
			problem(w, "invalid_request", 400)
			return
		}
	}
	state := r.Form.Get("state")
	cookie, err := r.Cookie(loginCookie)
	if err != nil || state == "" {
		problem(w, "invalid_request", 400)
		return
	}
	var pending pendingLogin
	err = s.store.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("hash = ? AND browser_hash = ? AND expires_at > ?", hash(state), hash(cookie.Value), time.Now()).First(&pending).Error; err != nil {
			return err
		}
		result := tx.Where("hash = ?", pending.Hash).Delete(&pendingLogin{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("state already used")
		}
		return nil
	})
	setCookie(w, loginCookie, "", -1)
	if err != nil {
		problem(w, "invalid_request", 400)
		return
	}
	if r.Form.Get("error") != "" || r.Form.Get("code") == "" {
		problem(w, "access_denied", 400)
		return
	}
	if iss := r.Form.Get("iss"); iss != "" && iss != s.cfg.UpstreamIssuer {
		problem(w, "invalid_request", 400)
		return
	}
	tokens, err := s.upstreamOAuth.Exchange(r.Context(), r.Form.Get("code"), oauth2.VerifierOption(pending.Verifier))
	if err != nil {
		problem(w, "access_denied", 400)
		return
	}
	raw, ok := tokens.Extra("id_token").(string)
	if !ok {
		problem(w, "access_denied", 400)
		return
	}
	verified, err := s.upstreamProvider.Verifier(&upstream.Config{ClientID: s.cfg.UpstreamClientID, SupportedSigningAlgs: []string{"RS256", "ES256"}}).Verify(r.Context(), raw)
	if err != nil || subtle.ConstantTimeCompare([]byte(verified.Nonce), []byte(pending.Nonce)) != 1 {
		problem(w, "access_denied", 400)
		return
	}
	var claims struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		EmailVerified   bool   `json:"email_verified"`
		SID             string `json:"sid"`
		AuthTime        int64  `json:"auth_time"`
		AuthorizedParty string `json:"azp"`
	}
	if err := verified.Claims(&claims); err != nil || verified.Subject == "" {
		problem(w, "access_denied", 400)
		return
	}
	if (len(verified.Audience) > 1 && claims.AuthorizedParty == "") || (claims.AuthorizedParty != "" && claims.AuthorizedParty != s.cfg.UpstreamClientID) {
		problem(w, "access_denied", 400)
		return
	}
	user, err := s.store.identityUser(r.Context(), s.cfg.UpstreamIssuer, verified.Subject, claims.Name, claims.Email, claims.EmailVerified)
	if err != nil {
		problem(w, "access_denied", 403)
		return
	}
	rawSession := randomToken()
	now := time.Now().UTC()
	browser := browserSession{Hash: hash(rawSession), UserUUID: user.UUID, UpstreamSubject: verified.Subject, UpstreamSID: claims.SID, CSRF: randomToken(), AuthTime: now, ExpiresAt: now.Add(12 * time.Hour)}
	if claims.AuthTime > 0 && claims.AuthTime <= now.Unix()+5 {
		browser.AuthTime = time.Unix(claims.AuthTime, 0)
	}
	if err := s.store.db.WithContext(r.Context()).Create(&browser).Error; err != nil {
		problem(w, "server_error", 500)
		return
	}
	setCookie(w, sessionCookie, rawSession, 12*60*60)
	destination := pending.ReturnTo
	if strings.HasPrefix(destination, "/authorize?") {
		continuation := randomToken()
		marker := pendingLogin{Hash: hash(continuation), BrowserHash: browser.Hash, ReturnTo: pending.ReturnTo, ExpiresAt: now.Add(5 * time.Minute)}
		if err := s.store.db.WithContext(r.Context()).Create(&marker).Error; err != nil {
			problem(w, "server_error", 500)
			return
		}
		u, _ := url.Parse(destination)
		q := u.Query()
		q.Set("transaction", continuation)
		u.RawQuery = q.Encode()
		destination = u.String()
	}
	http.Redirect(w, r, destination, http.StatusFound)
}
func (s *Server) backchannelLogout(w http.ResponseWriter, r *http.Request) {
	if s.upstreamProvider == nil {
		problem(w, "temporarily_unavailable", 503)
		return
	}
	if err := r.ParseForm(); err != nil || len(r.Form["logout_token"]) != 1 {
		problem(w, "invalid_request", 400)
		return
	}
	verified, err := s.upstreamProvider.Verifier(&upstream.Config{ClientID: s.cfg.UpstreamClientID, SupportedSigningAlgs: []string{"RS256", "ES256"}, SkipExpiryCheck: true}).Verify(r.Context(), r.Form.Get("logout_token"))
	if err != nil || (!verified.Expiry.IsZero() && !verified.Expiry.After(time.Now())) {
		problem(w, "invalid_request", 400)
		return
	}
	var claims struct {
		SID    string          `json:"sid"`
		JTI    string          `json:"jti"`
		Nonce  json.RawMessage `json:"nonce"`
		Events map[string]any  `json:"events"`
		IAT    int64           `json:"iat"`
	}
	if err := verified.Claims(&claims); err != nil || claims.JTI == "" || len(claims.Nonce) != 0 || claims.IAT < time.Now().Add(-5*time.Minute).Unix() || claims.IAT > time.Now().Add(30*time.Second).Unix() || claims.SID == "" && verified.Subject == "" {
		problem(w, "invalid_request", 400)
		return
	}
	event, ok := claims.Events["http://schemas.openid.net/event/backchannel-logout"]
	if !ok {
		problem(w, "invalid_request", 400)
		return
	}
	if eventMap, ok := event.(map[string]any); !ok || len(eventMap) != 0 {
		problem(w, "invalid_request", 400)
		return
	}
	replayExpiry := time.Unix(claims.IAT, 0).Add(5 * time.Minute)
	if !verified.Expiry.IsZero() && verified.Expiry.Before(replayExpiry) {
		replayExpiry = verified.Expiry
	}
	if err := s.store.SetClientAssertionJWT(r.Context(), "logout:"+s.cfg.UpstreamIssuer+":"+claims.JTI, replayExpiry); err != nil {
		problem(w, "invalid_request", 400)
		return
	}
	query := s.store.db.WithContext(r.Context()).Model(&browserSession{})
	if claims.SID != "" {
		query = query.Where("upstream_sid = ?", claims.SID)
	}
	if verified.Subject != "" {
		query = query.Where("upstream_subject = ?", verified.Subject)
	}
	if err := query.Update("revoked", true).Error; err != nil {
		problem(w, "server_error", 500)
		return
	}
	w.WriteHeader(200)
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html><html><head><meta charset="utf-8"><title>{{.Title}}</title></head><body><main><h1>{{.Title}}</h1>{{if .Message}}<p>{{.Message}}</p>{{end}}{{if .Consent}}<p>This application is unverified. Its name is supplied by its operator.</p><p>Client: {{.Client}}</p><p>Access: {{.Scopes}}</p><p>Resource: {{.Audience}}</p><form method="post" action="/consent">{{range .Fields}}<input type="hidden" name="{{.Name}}" value="{{.Value}}">{{end}}<input type="hidden" name="csrf" value="{{.CSRF}}"><button name="decision" value="approve">Allow access</button><button name="decision" value="deny">Deny</button></form>{{end}}{{if .Account}}<h2>Sessions</h2>{{range .Sessions}}<form method="post" action="/account"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="session_id" value="{{.Hash}}"><p>Signed in {{.AuthTime}}; expires {{.ExpiresAt}}</p><button name="action" value="revoke_session">Revoke session</button></form>{{end}}<h2>Applications</h2>{{range .Grants}}<form method="post" action="/account"><input type="hidden" name="csrf" value="{{$.CSRF}}"><input type="hidden" name="client_id" value="{{.ClientID}}"><p>{{.ClientID}}: {{.Scopes}}</p><button name="action" value="revoke_grant">Revoke access</button></form>{{end}}<p><a href="/logout">Sign out</a></p>{{end}}{{if .Logout}}<form method="post" action="/logout"><input type="hidden" name="csrf" value="{{.CSRF}}"><button>Sign out</button></form>{{end}}</main></body></html>`))

type field struct{ Name, Value string }
type page struct {
	Title, Message, Client, Scopes, Audience, CSRF string
	Consent, Account, Logout                       bool
	Fields                                         []field
	Sessions                                       []browserSession
	Grants                                         []consent
}

func renderPage(w http.ResponseWriter, r *http.Request, data page) {
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		sessions := []map[string]any{}
		for _, b := range data.Sessions {
			sessions = append(sessions, map[string]any{"session_id": b.Hash, "auth_time": b.AuthTime, "expires_at": b.ExpiresAt})
		}
		grants := []map[string]any{}
		for _, g := range data.Grants {
			grants = append(grants, map[string]any{"client_id": g.ClientID, "scope": g.Scopes, "resource": g.Audience, "created_at": g.CreatedAt})
		}
		fields := map[string]string{}
		for _, f := range data.Fields {
			fields[f.Name] = f.Value
		}
		writeJSON(w, 200, map[string]any{"title": data.Title, "message": data.Message, "client_id": data.Client, "scope": data.Scopes, "resource": data.Audience, "csrf": data.CSRF, "fields": fields, "sessions": sessions, "grants": grants})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTemplate.Execute(w, data)
}
func (s *Server) renderConsent(w http.ResponseWriter, r *http.Request, m *ClientMetadata, scopes []string, audience, csrf string, form url.Values) {
	data := page{Title: "Allow " + m.ClientName + " to access your account?", Client: m.ClientID, Consent: true, Scopes: strings.Join(scopes, " "), Audience: audience, CSRF: csrf}
	for k, vs := range form {
		if k == "csrf" || k == "decision" {
			continue
		}
		for _, v := range vs {
			data.Fields = append(data.Fields, field{Name: k, Value: v})
		}
	}
	renderPage(w, r, data)
}
func (s *Server) account(w http.ResponseWriter, r *http.Request) {
	browser, user, err := s.browser(r)
	if err != nil {
		s.beginLogin(w, r, "/account")
		return
	}
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil || !s.validCSRF(r, browser) {
			problem(w, "invalid_request", 403)
			return
		}
		switch r.Form.Get("action") {
		case "revoke_session":
			err = s.store.db.WithContext(r.Context()).Model(&browserSession{}).Where("hash = ? AND user_uuid = ?", r.Form.Get("session_id"), user.UUID).Update("revoked", true).Error
		case "revoke_grant":
			err = s.store.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&consent{}).Where("user_uuid = ? AND client_id = ?", user.UUID, r.Form.Get("client_id")).Update("revoked", true).Error; err != nil {
					return err
				}
				return tx.Model(&oauthRecord{}).Where("user_uuid = ? AND client_id = ?", user.UUID, r.Form.Get("client_id")).Update("active", false).Error
			})
		default:
			problem(w, "invalid_request", 400)
			return
		}
		if err != nil {
			problem(w, "server_error", 500)
			return
		}
		http.Redirect(w, r, "/account", http.StatusSeeOther)
		return
	}
	data := page{Title: "Your account", Message: user.Name, Account: true, CSRF: browser.CSRF}
	if err := s.store.db.WithContext(r.Context()).Where("user_uuid = ? AND revoked = ? AND expires_at > ?", user.UUID, false, time.Now()).Find(&data.Sessions).Error; err != nil {
		problem(w, "server_error", 500)
		return
	}
	if err := s.store.db.WithContext(r.Context()).Where("user_uuid = ? AND revoked = ?", user.UUID, false).Find(&data.Grants).Error; err != nil {
		problem(w, "server_error", 500)
		return
	}
	renderPage(w, r, data)
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	browser, _, err := s.browser(r)
	if err != nil {
		renderPage(w, r, page{Title: "Signed out"})
		return
	}
	if r.Method == "GET" {
		renderPage(w, r, page{Title: "Sign out?", Logout: true, CSRF: browser.CSRF})
		return
	}
	if err := r.ParseForm(); err != nil || !s.validCSRF(r, browser) {
		problem(w, "invalid_request", 403)
		return
	}
	if err := s.store.db.WithContext(r.Context()).Model(&browserSession{}).Where("hash = ?", browser.Hash).Update("revoked", true).Error; err != nil {
		problem(w, "server_error", 500)
		return
	}
	setCookie(w, sessionCookie, "", -1)
	renderPage(w, r, page{Title: "Signed out"})
}

func issuerOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
