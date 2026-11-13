package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/token/jwt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"within.website/x/cmd/iamd/models"
)

// Every durable OIDC row uses a separate table namespace from IAM credentials.
type identity struct {
	ID            uint   `gorm:"primaryKey"`
	Issuer        string `gorm:"uniqueIndex:oidc_identity"`
	Subject       string `gorm:"uniqueIndex:oidc_identity"`
	UserUUID      string `gorm:"index"`
	Name          string
	Email         string
	EmailVerified bool
}

func (identity) TableName() string { return "oidc_identities" }

type browserSession struct {
	Hash            string `gorm:"primaryKey"`
	UserUUID        string `gorm:"index"`
	UpstreamSubject string `gorm:"index"`
	UpstreamSID     string `gorm:"index"`
	CSRF            string
	AuthTime        time.Time
	ExpiresAt       time.Time
	Revoked         bool
}

func (browserSession) TableName() string { return "oidc_browser_sessions" }

type pendingLogin struct {
	Hash        string `gorm:"primaryKey"`
	BrowserHash string
	Nonce       string
	Verifier    string
	ReturnTo    string
	ExpiresAt   time.Time
}

func (pendingLogin) TableName() string { return "oidc_pending_logins" }

type oauthRecord struct {
	Kind        string `gorm:"primaryKey"`
	Hash        string `gorm:"primaryKey"`
	RequestID   string `gorm:"index"`
	UserUUID    string `gorm:"index"`
	BrowserHash string `gorm:"index"`
	ClientID    string `gorm:"index"`
	Data        []byte
	Active      bool
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

func (oauthRecord) TableName() string { return "oidc_oauth_records" }

type consent struct {
	UserUUID     string `gorm:"primaryKey"`
	ClientID     string `gorm:"primaryKey"`
	Scopes       string
	Audience     string
	MetadataHash string
	CreatedAt    time.Time
	Revoked      bool
}

func (consent) TableName() string { return "oidc_consents" }

type clientRecord struct {
	ID         string `gorm:"primaryKey"`
	Metadata   []byte
	SecretHash []byte
	Disabled   bool
	CreatedAt  time.Time
}

func (clientRecord) TableName() string { return "oidc_clients" }

type resource struct {
	Audience string `gorm:"primaryKey" json:"audience"`
	Name     string `json:"name"`
	Scopes   string `json:"scopes"`
}

func (resource) TableName() string { return "oidc_resources" }

type keyRecord struct {
	ID        string `gorm:"primaryKey"`
	DER       []byte
	CreatedAt time.Time
	RetiredAt *time.Time
}

func (keyRecord) TableName() string { return "oidc_keys" }

type secretRecord struct {
	Name  string `gorm:"primaryKey"`
	Value []byte
}

func (secretRecord) TableName() string { return "oidc_secrets" }

type jtiRecord struct {
	Hash      string `gorm:"primaryKey"`
	ExpiresAt time.Time
}

func (jtiRecord) TableName() string { return "oidc_jtis" }

type rateRecord struct {
	Key    string `gorm:"primaryKey"`
	Window int64
	Count  int
}

func (rateRecord) TableName() string { return "oidc_rate_limits" }

type store struct {
	issuer string
	db     *gorm.DB
	dao    *models.DAO
}

func newStore(dao *models.DAO) (*store, error) {
	db := dao.DB().Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	// SQLite read-to-write upgrades cannot wait behind another transaction.
	// One replica uses one connection, serializing all IAM/OIDC transactions
	// and preventing both duplicate identity provisioning and busy-lock races.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get OIDC SQL connection: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := db.AutoMigrate(&identity{}, &browserSession{}, &pendingLogin{}, &oauthRecord{}, &consent{}, &clientRecord{}, &resource{}, &keyRecord{}, &secretRecord{}, &jtiRecord{}, &rateRecord{}); err != nil {
		return nil, fmt.Errorf("migrate OIDC: %w", err)
	}
	return &store{db: db, dao: dao}, nil
}
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func hash(v string) string {
	h := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
func (s *store) identityUser(ctx context.Context, iss, sub, name, email string, verified bool) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ident identity
		err := tx.Where("issuer = ? AND subject = ?", iss, sub).First(&ident).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			user = models.User{UUID: id.String(), Name: name}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			ident = identity{Issuer: iss, Subject: sub, UserUUID: user.UUID, Name: name, Email: email, EmailVerified: verified}
			return tx.Create(&ident).Error
		}
		if err != nil {
			return err
		}
		// Scoped query refuses disabled users. Profile refresh never clears disable state.
		if err := tx.Where("uuid = ?", ident.UserUUID).First(&user).Error; err != nil {
			return err
		}
		if err := tx.Model(&ident).Updates(map[string]any{"name": name, "email": email, "email_verified": verified}).Error; err != nil {
			return err
		}
		return tx.Model(&user).Update("name", name).Error
	})
	return &user, err
}

type session struct {
	*openid.DefaultSession
	AccessClaims  *jwt.JWTClaims `json:"access_claims"`
	AccessHeaders *jwt.Headers   `json:"access_headers"`
	BrowserHash   string         `json:"browser_hash"`
	FamilyStarted time.Time      `json:"family_started"`
	MetadataHash  string         `json:"metadata_hash"`
}

func newSession(subject, browser string, authTime time.Time) *session {
	o := openid.NewDefaultSession()
	o.Subject = subject
	o.Claims.Subject = subject
	o.Claims.AuthTime = authTime
	return &session{DefaultSession: o, AccessClaims: &jwt.JWTClaims{Subject: subject}, AccessHeaders: &jwt.Headers{Extra: map[string]any{"typ": "at+JWT"}}, BrowserHash: browser, FamilyStarted: time.Now().UTC()}
}
func (s *session) GetJWTClaims() jwt.JWTClaimsContainer { return s.AccessClaims }
func (s *session) GetJWTHeader() *jwt.Headers           { return s.AccessHeaders }
func (s *session) Clone() fosite.Session {
	b, _ := json.Marshal(s)
	out := newSession("", "", time.Time{})
	_ = json.Unmarshal(b, out)
	return out
}

type storedRequest struct {
	Request *fosite.Request `json:"request"`
}

func encodeRequest(req fosite.Requester) ([]byte, error) {
	copy := fosite.NewRequest()
	copy.Merge(req)
	copy.ID = req.GetID()
	// A fresh copy prevents accidental retention of token endpoint credentials.
	copy.Form = req.Sanitize([]string{"redirect_uri", "code_challenge", "code_challenge_method", "nonce", "prompt", "max_age", "resource"}).GetRequestForm()
	return json.Marshal(storedRequest{Request: copy})
}
func decodeRequest(data []byte) (fosite.Requester, error) {
	req := fosite.NewRequest()
	req.Client = &fosite.DefaultOpenIDConnectClient{DefaultClient: &fosite.DefaultClient{}}
	req.Session = newSession("", "", time.Time{})
	out := storedRequest{Request: req}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out.Request, nil
}

type txKey struct{}

func (s *store) dbFor(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return s.db.WithContext(ctx)
}
func (s *store) BeginTX(ctx context.Context) (context.Context, error) {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return ctx, tx.Error
	}
	return context.WithValue(ctx, txKey{}, tx), nil
}
func (s *store) Commit(ctx context.Context) error   { return s.dbFor(ctx).Commit().Error }
func (s *store) Rollback(ctx context.Context) error { return s.dbFor(ctx).Rollback().Error }
func (s *store) put(ctx context.Context, kind, key string, req fosite.Requester) error {
	data, err := encodeRequest(req)
	if err != nil {
		return err
	}
	sess, ok := req.GetSession().(*session)
	if !ok {
		return errors.New("unsupported OIDC session")
	}
	expires := time.Now().Add(5 * time.Minute)
	if kind == "access" {
		expires = req.GetSession().GetExpiresAt(fosite.AccessToken)
	}
	if kind == "refresh" {
		expires = time.Now().Add(30 * 24 * time.Hour)
		absolute := sess.FamilyStarted.Add(90 * 24 * time.Hour)
		if absolute.Before(expires) {
			expires = absolute
		}
	}
	rec := oauthRecord{Kind: kind, Hash: hash(key), RequestID: req.GetID(), UserUUID: sess.Subject, BrowserHash: sess.BrowserHash, ClientID: req.GetClient().GetID(), Data: data, Active: true, ExpiresAt: expires}
	return s.dbFor(ctx).Create(&rec).Error
}
func (s *store) get(ctx context.Context, kind, key string) (fosite.Requester, error) {
	var rec oauthRecord
	if err := s.dbFor(ctx).Where("kind = ? AND hash = ?", kind, hash(key)).First(&rec).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fosite.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	req, err := decodeRequest(rec.Data)
	if err != nil {
		return nil, err
	}
	if !rec.Active {
		if kind == "code" {
			return req, fosite.ErrInvalidatedAuthorizeCode
		}
		if kind == "refresh" {
			refreshReuse.Inc()
		}
		return req, fosite.ErrInactiveToken
	}
	if time.Now().After(rec.ExpiresAt) {
		return nil, fosite.ErrNotFound
	}
	var user models.User
	if err := s.dbFor(ctx).Where("uuid = ?", rec.UserUUID).First(&user).Error; err != nil {
		return nil, fosite.ErrNotFound
	}
	var browser browserSession
	err = s.dbFor(ctx).Where("hash = ?", rec.BrowserHash).First(&browser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fosite.ErrNotFound
	}
	if err == nil && browser.Revoked {
		return nil, fosite.ErrNotFound
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return req, nil
}
func (s *store) deactivate(ctx context.Context, kind, key string) error {
	return s.dbFor(ctx).Model(&oauthRecord{}).Where("kind = ? AND hash = ?", kind, hash(key)).Update("active", false).Error
}
func (s *store) CreateAuthorizeCodeSession(ctx context.Context, key string, r fosite.Requester) error {
	return s.put(ctx, "code", key, r)
}
func (s *store) GetAuthorizeCodeSession(ctx context.Context, key string, _ fosite.Session) (fosite.Requester, error) {
	return s.get(ctx, "code", key)
}
func (s *store) InvalidateAuthorizeCodeSession(ctx context.Context, key string) error {
	result := s.dbFor(ctx).Model(&oauthRecord{}).Where("kind = ? AND hash = ? AND active = ?", "code", hash(key), true).Update("active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fosite.ErrInvalidatedAuthorizeCode
	}
	return nil
}
func (s *store) CreatePKCERequestSession(ctx context.Context, key string, r fosite.Requester) error {
	return s.put(ctx, "pkce", key, r)
}
func (s *store) GetPKCERequestSession(ctx context.Context, key string, _ fosite.Session) (fosite.Requester, error) {
	return s.get(ctx, "pkce", key)
}
func (s *store) DeletePKCERequestSession(ctx context.Context, key string) error {
	return s.deactivate(ctx, "pkce", key)
}
func (s *store) CreateOpenIDConnectSession(ctx context.Context, key string, r fosite.Requester) error {
	return s.put(ctx, "openid", key, r)
}
func (s *store) GetOpenIDConnectSession(ctx context.Context, key string, _ fosite.Requester) (fosite.Requester, error) {
	return s.get(ctx, "openid", key)
}
func (s *store) DeleteOpenIDConnectSession(ctx context.Context, key string) error {
	return s.deactivate(ctx, "openid", key)
}
func (s *store) CreateAccessTokenSession(ctx context.Context, key string, r fosite.Requester) error {
	return s.put(ctx, "access", key, r)
}
func (s *store) GetAccessTokenSession(ctx context.Context, key string, _ fosite.Session) (fosite.Requester, error) {
	return s.get(ctx, "access", key)
}
func (s *store) DeleteAccessTokenSession(ctx context.Context, key string) error {
	return s.deactivate(ctx, "access", key)
}
func (s *store) CreateRefreshTokenSession(ctx context.Context, key, _ string, r fosite.Requester) error {
	return s.put(ctx, "refresh", key, r)
}
func (s *store) GetRefreshTokenSession(ctx context.Context, key string, _ fosite.Session) (fosite.Requester, error) {
	return s.get(ctx, "refresh", key)
}
func (s *store) DeleteRefreshTokenSession(ctx context.Context, key string) error {
	return s.deactivate(ctx, "refresh", key)
}
func (s *store) RevokeRefreshToken(ctx context.Context, id string) error {
	return s.dbFor(ctx).Model(&oauthRecord{}).Where("kind = ? AND request_id = ?", "refresh", id).Update("active", false).Error
}
func (s *store) RevokeAccessToken(ctx context.Context, id string) error {
	return s.dbFor(ctx).Model(&oauthRecord{}).Where("kind = ? AND request_id = ?", "access", id).Update("active", false).Error
}
func (s *store) RotateRefreshToken(ctx context.Context, id, key string) error {
	result := s.dbFor(ctx).Model(&oauthRecord{}).Where("kind = ? AND hash = ? AND active = ?", "refresh", hash(key), true).Update("active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fosite.ErrInactiveToken
	}
	return s.RevokeAccessToken(ctx, id)
}
func (s *store) ClientAssertionJWTValid(ctx context.Context, jti string) error {
	var n int64
	if err := s.dbFor(ctx).Model(&jtiRecord{}).Where("hash = ? AND expires_at > ?", hash(jti), time.Now()).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return fosite.ErrJTIKnown
	}
	return nil
}
func (s *store) SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) error {
	// Atomic unique insert protects against assertion replay even with concurrent requests.
	result := s.dbFor(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&jtiRecord{Hash: hash(jti), ExpiresAt: exp})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fosite.ErrJTIKnown
	}
	return nil
}
