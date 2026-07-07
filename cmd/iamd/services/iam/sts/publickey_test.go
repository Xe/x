package sts

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"testing"
	"time"

	"github.com/twitchtv/twirp"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4a"
)

func TestGetPublicKey(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns the derived public key", func(t *testing.T) {
		s, _, akid, secret := newSigningKeysTest(t)
		resp, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: akid})
		if err != nil {
			t.Fatalf("GetPublicKey: %v", err)
		}

		parsed, err := x509.ParsePKIXPublicKey(resp.GetPublicKey())
		if err != nil {
			t.Fatalf("response public key does not parse: %v", err)
		}
		pub, ok := parsed.(*ecdsa.PublicKey)
		if !ok {
			t.Fatalf("public key is %T, want *ecdsa.PublicKey", parsed)
		}
		priv, err := sigv4a.DeriveKeyPair(akid, secret)
		if err != nil {
			t.Fatalf("DeriveKeyPair: %v", err)
		}
		if !pub.Equal(&priv.PublicKey) {
			t.Error("response public key does not match the key derived from the stored secret")
		}

		if got := resp.GetIdentity().GetDisplayName(); got != "tester" {
			t.Errorf("display_name = %q, want tester", got)
		}
		if resp.GetIdentity().GetAccessKeyId() != akid {
			t.Errorf("identity access_key_id = %q, want %q", resp.GetIdentity().GetAccessKeyId(), akid)
		}
		if got := resp.GetCacheUntil().AsTime(); !got.Equal(fixedNow.Add(5 * time.Minute)) {
			t.Errorf("cache_until = %v, want now+5m", got)
		}
	})

	t.Run("unknown key is NOT_FOUND", func(t *testing.T) {
		s, _, _, _ := newSigningKeysTest(t)
		_, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: "AKIDNOPE"})
		wantTwirpCode(t, err, twirp.NotFound)
	})

	t.Run("disabled key is PERMISSION_DENIED", func(t *testing.T) {
		s, dao, akid, _ := newSigningKeysTest(t)
		if err := dao.DisableKey(ctx, akid, "test", ""); err != nil {
			t.Fatalf("DisableKey: %v", err)
		}
		_, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: akid})
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
		_, err = s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: akid})
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("missing access_key_id is INVALID_ARGUMENT", func(t *testing.T) {
		s, _, _, _ := newSigningKeysTest(t)
		_, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{})
		wantTwirpCode(t, err, twirp.InvalidArgument)
	})
}
