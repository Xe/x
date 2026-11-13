package oidc

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"
	"within.website/x/cmd/iamd/models"
	"within.website/x/web/middleware/authctx"
)

func (s *Server) admin(w http.ResponseWriter, r *http.Request) {
	// The Connect layer verifies SigV4 before attaching this identity. Browser
	// sessions never authorize IAM administration.
	user, ok := authctx.User(r.Context())
	if !ok || user == nil || !user.GetIsAdmin() {
		problem(w, "access_denied", 403)
		return
	}
	switch r.URL.Path {
	case "/admin/resources":
		if !method(w, r, "GET", "POST") {
			return
		}
		if r.Method == "GET" {
			var rows []resource
			if err := s.store.db.WithContext(r.Context()).Find(&rows).Error; err != nil {
				problem(w, "server_error", 500)
				return
			}
			writeJSON(w, 200, map[string]any{"resources": rows})
			return
		}
		var res resource
		if err := readJSON(r, &res); err != nil {
			problem(w, "invalid_request", 400)
			return
		}
		audience, err := url.Parse(res.Audience)
		if err != nil || !audience.IsAbs() || audience.Fragment != "" || audience.User != nil || res.Audience == s.cfg.Issuer+"/userinfo" || len(res.Name) > 200 || len(res.Scopes) > 4096 {
			problem(w, "invalid_request", 400)
			return
		}
		scopes := strings.Fields(res.Scopes)
		if len(scopes) == 0 {
			problem(w, "invalid_scope", 400)
			return
		}
		for _, scope := range scopes {
			if slices.Contains(baseScopes, scope) || strings.ContainsAny(scope, "\"\\") || len(scope) > 200 {
				problem(w, "invalid_scope", 400)
				return
			}
		}
		res.Scopes = strings.Join(scopes, " ")
		if err := s.store.db.WithContext(r.Context()).Save(&res).Error; err != nil {
			problem(w, "server_error", 500)
			return
		}
		writeJSON(w, 200, res)
	case "/admin/clients":
		if !method(w, r, "GET", "POST") {
			return
		}
		if r.Method == "GET" {
			var rows []clientRecord
			if err := s.store.db.WithContext(r.Context()).Find(&rows).Error; err != nil {
				problem(w, "server_error", 500)
				return
			}
			out := []map[string]any{}
			for _, row := range rows {
				var metadata any
				_ = json.Unmarshal(row.Metadata, &metadata)
				out = append(out, map[string]any{"client_id": row.ID, "disabled": row.Disabled, "metadata": metadata, "created_at": row.CreatedAt})
			}
			writeJSON(w, 200, map[string]any{"clients": out})
			return
		}
		var body struct {
			ClientID string `json:"client_id"`
			Disabled *bool  `json:"disabled"`
		}
		if err := readJSON(r, &body); err != nil || body.ClientID == "" {
			problem(w, "invalid_request", 400)
			return
		}
		disabled := true
		if body.Disabled != nil {
			disabled = *body.Disabled
		}
		err := s.store.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
			var row clientRecord
			if err := tx.Where("id = ?", body.ClientID).First(&row).Error; err == gorm.ErrRecordNotFound && strings.HasPrefix(body.ClientID, "https://") {
				row = clientRecord{ID: body.ClientID, Disabled: disabled}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			if err := tx.Model(&clientRecord{}).Where("id = ?", body.ClientID).Update("disabled", disabled).Error; err != nil {
				return err
			}
			if disabled {
				if err := tx.Model(&consent{}).Where("client_id = ?", body.ClientID).Update("revoked", true).Error; err != nil {
					return err
				}
				return tx.Model(&oauthRecord{}).Where("client_id = ?", body.ClientID).Update("active", false).Error
			}
			return nil
		})
		if err != nil {
			problem(w, "invalid_request", 400)
			return
		}
		writeJSON(w, 200, map[string]any{"client_id": body.ClientID, "disabled": disabled})
	case "/admin/identities":
		if !method(w, r, "POST") {
			return
		}
		var body struct {
			Issuer  string `json:"issuer"`
			Subject string `json:"subject"`
			UserID  string `json:"user_id"`
		}
		if err := readJSON(r, &body); err != nil || body.Issuer != s.cfg.UpstreamIssuer || body.Subject == "" || body.UserID == "" {
			problem(w, "invalid_request", 400)
			return
		}
		err := s.store.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
			var user models.User
			if err := tx.Where("uuid = ?", body.UserID).First(&user).Error; err != nil {
				return err
			}
			// Linking is explicit and never steals an existing identity. It creates no
			// keys and changes no local administrator or disabled flags.
			return tx.Create(&identity{Issuer: body.Issuer, Subject: body.Subject, UserUUID: body.UserID}).Error
		})
		if err != nil {
			problem(w, "invalid_request", 400)
			return
		}
		writeJSON(w, 201, map[string]any{"issuer": body.Issuer, "subject": body.Subject, "user_id": body.UserID})
	case "/admin/keys/rotate":
		if !method(w, r, "POST") {
			return
		}
		if err := s.rotateKey(r.Context()); err != nil {
			problem(w, "server_error", 500)
			return
		}
		s.keyMu.RLock()
		id := s.currentKey.ID
		s.keyMu.RUnlock()
		writeJSON(w, 200, map[string]any{"kid": id, "rotated_at": time.Now().UTC()})
	default:
		http.NotFound(w, r)
	}
}
