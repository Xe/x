package oidc

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ory/fosite"
	"gorm.io/gorm"
	"within.website/x/cmd/iamd/models"
)

func testStore(t *testing.T) *store {
	t.Helper()
	dao, err := models.Open(filepath.Join(t.TempDir(), "iam.db"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := newStore(dao)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestIdentityStableAndDisabled(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, err := s.identityUser(ctx, "https://idp.xeserv.us", "subject", "Alice", "alice@example.com", true)
	if err != nil {
		t.Fatal(err)
	}
	if u.IsAdmin {
		t.Fatal("upstream identity became admin")
	}
	again, err := s.identityUser(ctx, "https://idp.xeserv.us", "subject", "Alice II", "new@example.com", false)
	if err != nil {
		t.Fatal(err)
	}
	if again.UUID != u.UUID {
		t.Fatal("profile update changed subject")
	}
	if err := s.dao.DisableUser(ctx, u.UUID, "disabled"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.identityUser(ctx, "https://idp.xeserv.us", "subject", "Alice", "", false); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("disabled login error = %v", err)
	}
}
func TestOAuthStorageRestartAndRotation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, err := s.dao.CreateUser(ctx, "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.db.Create(&browserSession{Hash: "browser", UserUUID: u.UUID, ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	req := fosite.NewRequest()
	req.Client = &fosite.DefaultClient{ID: "client"}
	req.SetSession(newSession(u.UUID, "browser", time.Now()))
	req.GrantScope("openid")
	req.GrantScope("offline_access")
	if err := s.CreateRefreshTokenSession(ctx, "signature", "access", req); err != nil {
		t.Fatal(err)
	}
	restarted, err := newStore(s.dao)
	if err != nil {
		t.Fatal(err)
	}
	got, err := restarted.GetRefreshTokenSession(ctx, "signature", newSession("", "", time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	if got.GetSession().GetSubject() != u.UUID {
		t.Fatal("subject did not survive restart")
	}
	if err := restarted.RotateRefreshToken(ctx, got.GetID(), "signature"); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.GetRefreshTokenSession(ctx, "signature", nil); !errors.Is(err, fosite.ErrInactiveToken) {
		t.Fatalf("reused refresh error = %v", err)
	}
}
