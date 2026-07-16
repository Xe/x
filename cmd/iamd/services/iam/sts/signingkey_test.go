package sts

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/twitchtv/twirp"

	"within.website/x/cmd/iamd/models"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

const (
	skRegion  = "us-east-1"
	skService = "iam"
)

// fixedNow is an arbitrary instant mid-day UTC so the ±1 day issuance window
// is unambiguous.
var fixedNow = time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)

// newDAO opens a fresh SQLite-backed DAO in a temp directory for testing.
func newDAO(t *testing.T) *models.DAO {
	t.Helper()
	dao, err := models.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return dao
}

func newSigningKeysTest(t *testing.T) (*SigningKeys, *models.DAO, string, string) {
	t.Helper()
	dao := newDAO(t)
	u, err := dao.CreateUser(context.Background(), "tester")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	k, err := dao.CreateKey(context.Background(), u, "test key")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	s := NewSigningKeys(dao, 5*time.Minute)
	s.Now = func() time.Time { return fixedNow }
	return s, dao, k.AccessKeyID, k.SecretAccessKey
}

func validReq(akid string) *stsv1.GetSigningKeyRequest {
	return &stsv1.GetSigningKeyRequest{
		AccessKeyId: akid,
		Date:        fixedNow.Format("20060102"),
		Region:      skRegion,
		Service:     skService,
	}
}

func wantTwirpCode(t *testing.T, err error, want twirp.ErrorCode) {
	t.Helper()
	var te twirp.Error
	if !errors.As(err, &te) {
		t.Fatalf("err = %v, want twirp %s", err, want)
	}
	if te.Code() != want {
		t.Fatalf("code = %s, want %s", te.Code(), want)
	}
}

func TestGetSigningKey(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		s, _, akid, secret := newSigningKeysTest(t)
		resp, err := s.GetSigningKey(ctx, validReq(akid))
		if err != nil {
			t.Fatalf("GetSigningKey: %v", err)
		}
		want := sigv4.DeriveSigningKey(secret, fixedNow.Format("20060102"), skRegion, skService)
		if !bytes.Equal(resp.GetSigningKey(), want) {
			t.Error("derived key mismatch")
		}
		if got := resp.GetIdentity().GetDisplayName(); got != "tester" {
			t.Errorf("display_name = %q, want tester", got)
		}
		if resp.GetIdentity().GetPrincipalId() == "" {
			t.Error("principal_id empty")
		}
		if resp.GetIdentity().GetAccessKeyId() != akid {
			t.Errorf("identity access_key_id = %q, want %q", resp.GetIdentity().GetAccessKeyId(), akid)
		}

		nva := resp.GetNotValidAfter().AsTime()
		wantNVA := time.Date(2026, 7, 7, 0, 15, 0, 0, time.UTC) // end of UTC day + 15m skew
		if !nva.Equal(wantNVA) {
			t.Errorf("not_valid_after = %v, want %v", nva, wantNVA)
		}
		cu := resp.GetCacheUntil().AsTime()
		if !cu.Equal(fixedNow.Add(5 * time.Minute)) {
			t.Errorf("cache_until = %v, want now+5m", cu)
		}
		if cu.After(nva) {
			t.Error("cache_until exceeds not_valid_after")
		}
	})

	t.Run("cache_until clamped to not_valid_after", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		// 5 minutes before the validity bound, cache_until must clamp.
		s.Now = func() time.Time { return time.Date(2026, 7, 7, 0, 12, 0, 0, time.UTC) }
		req := validReq(akid)
		req.Date = "20260706" // still yesterday's scope, inside the ±1 day window
		resp, err := s.GetSigningKey(ctx, req)
		if err != nil {
			t.Fatalf("GetSigningKey: %v", err)
		}
		wantNVA := time.Date(2026, 7, 7, 0, 15, 0, 0, time.UTC)
		if !resp.GetCacheUntil().AsTime().Equal(wantNVA) {
			t.Errorf("cache_until = %v, want clamped to %v", resp.GetCacheUntil().AsTime(), wantNVA)
		}
	})

	t.Run("unknown key is NOT_FOUND", func(t *testing.T) {
		s, _, _, _ := newSigningKeysTest(t)
		_, err := s.GetSigningKey(ctx, validReq("AKIDNOPE"))
		wantTwirpCode(t, err, twirp.NotFound)
	})

	t.Run("disabled key is PERMISSION_DENIED", func(t *testing.T) {
		s, dao, akid, _ := newSigningKeysTest(t)
		if err := dao.DisableKey(ctx, akid, "test", ""); err != nil {
			t.Fatalf("DisableKey: %v", err)
		}
		_, err := s.GetSigningKey(ctx, validReq(akid))
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("disabled user is PERMISSION_DENIED", func(t *testing.T) {
		s, dao, akid, _ := newSigningKeysTest(t)
		us, err := dao.ListUsers(ctx, 10, 0)
		if err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		if err := dao.DisableUser(ctx, us[0].UUID, "test"); err != nil {
			t.Fatalf("DisableUser: %v", err)
		}
		_, err = s.GetSigningKey(ctx, validReq(akid))
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("date outside issuance window is PERMISSION_DENIED", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		req := validReq(akid)
		req.Date = "20260101"
		_, err := s.GetSigningKey(ctx, req)
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("malformed date is INVALID_ARGUMENT", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		for _, bad := range []string{"", "2026-07-06", "20261340", "garbage!"} {
			req := validReq(akid)
			req.Date = bad
			_, err := s.GetSigningKey(ctx, req)
			wantTwirpCode(t, err, twirp.InvalidArgument)
		}
	})

	t.Run("missing fields are INVALID_ARGUMENT", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		for name, mut := range map[string]func(*stsv1.GetSigningKeyRequest){
			"access_key_id": func(r *stsv1.GetSigningKeyRequest) { r.AccessKeyId = "" },
			"region":        func(r *stsv1.GetSigningKeyRequest) { r.Region = "" },
			"service":       func(r *stsv1.GetSigningKeyRequest) { r.Service = "" },
		} {
			req := validReq(akid)
			mut(req)
			_, err := s.GetSigningKey(ctx, req)
			if err == nil {
				t.Fatalf("%s: missing field accepted", name)
			}
			wantTwirpCode(t, err, twirp.InvalidArgument)
		}
	})
}
