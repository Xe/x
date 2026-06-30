package models

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"gorm.io/gorm"
)

func openTestDAO(t *testing.T) *DAO {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return d
}

func mustCreateUser(t *testing.T, d *DAO, name string) *User {
	t.Helper()
	u, err := d.CreateUser(context.Background(), name)
	if err != nil {
		t.Fatalf("CreateUser(%q): %v", name, err)
	}
	return u
}

func mustCreateKey(t *testing.T, d *DAO, u *User, comment string) *Key {
	t.Helper()
	k, err := d.CreateKey(context.Background(), u, comment)
	if err != nil {
		t.Fatalf("CreateKey(%q): %v", comment, err)
	}
	return k
}

func TestGetUser(t *testing.T) {
	d := openTestDAO(t)
	u := mustCreateUser(t, d, "alice")

	cases := []struct {
		name    string
		userID  string
		want    string // expected UUID when no error is expected
		wantErr error
	}{
		{name: "existing user", userID: u.UUID, want: u.UUID},
		{name: "missing user", userID: "does-not-exist", wantErr: gorm.ErrRecordNotFound},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetUser(context.Background(), tt.userID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetUser err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && got.UUID != tt.want {
				t.Errorf("GetUser UUID = %q, want %q", got.UUID, tt.want)
			}
		})
	}
}

func TestGetUserByAccessKeyID(t *testing.T) {
	d := openTestDAO(t)
	owner := mustCreateUser(t, d, "owner")
	key := mustCreateKey(t, d, owner, "active")

	disabledOwner := mustCreateUser(t, d, "disabled-owner")
	disabledKey := mustCreateKey(t, d, disabledOwner, "disabled")
	if err := d.DisableKey(context.Background(), disabledKey.AccessKeyID, "test", ""); err != nil {
		t.Fatalf("DisableKey setup: %v", err)
	}

	cases := []struct {
		name        string
		accessKeyID string
		want        string // expected user UUID when no error is expected
		wantErr     error
	}{
		{name: "resolves to owning user", accessKeyID: key.AccessKeyID, want: owner.UUID},
		{name: "unknown key", accessKeyID: "AKIDNOPE", wantErr: gorm.ErrRecordNotFound},
		{name: "disabled key is not found", accessKeyID: disabledKey.AccessKeyID, wantErr: gorm.ErrRecordNotFound},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetUserByAccessKeyID(context.Background(), tt.accessKeyID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetUserByAccessKeyID err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && got.UUID != tt.want {
				t.Errorf("user UUID = %q, want %q", got.UUID, tt.want)
			}
		})
	}
}

func TestSecretFor(t *testing.T) {
	d := openTestDAO(t)
	owner := mustCreateUser(t, d, "owner")
	key := mustCreateKey(t, d, owner, "active")

	disabledOwner := mustCreateUser(t, d, "disabled-owner")
	disabledKey := mustCreateKey(t, d, disabledOwner, "disabled")
	if err := d.DisableKey(context.Background(), disabledKey.AccessKeyID, "test", ""); err != nil {
		t.Fatalf("DisableKey setup: %v", err)
	}

	cases := []struct {
		name        string
		accessKeyID string
		want        string // expected secret when no error is expected
		wantErr     error
	}{
		{name: "resolves active key secret", accessKeyID: key.AccessKeyID, want: key.SecretAccessKey},
		{name: "unknown key", accessKeyID: "AKIDNOPE", wantErr: gorm.ErrRecordNotFound},
		{name: "disabled key is not found", accessKeyID: disabledKey.AccessKeyID, wantErr: gorm.ErrRecordNotFound},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.SecretFor(context.Background(), tt.accessKeyID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SecretFor err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("secret = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDisableKey(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T, d *DAO) (keyID, userID string)
		wantErr error
	}{
		{
			name: "owner can disable own key",
			setup: func(t *testing.T, d *DAO) (string, string) {
				u := mustCreateUser(t, d, "owner")
				k := mustCreateKey(t, d, u, "k")
				return k.AccessKeyID, u.UUID
			},
		},
		{
			name: "non-owner scope is rejected",
			setup: func(t *testing.T, d *DAO) (string, string) {
				owner := mustCreateUser(t, d, "owner")
				other := mustCreateUser(t, d, "other")
				k := mustCreateKey(t, d, owner, "k")
				return k.AccessKeyID, other.UUID
			},
			wantErr: gorm.ErrRecordNotFound,
		},
		{
			name: "no scope disables regardless of owner",
			setup: func(t *testing.T, d *DAO) (string, string) {
				u := mustCreateUser(t, d, "owner")
				k := mustCreateKey(t, d, u, "k")
				return k.AccessKeyID, ""
			},
		},
		{
			name: "unknown key id is not found",
			setup: func(t *testing.T, d *DAO) (string, string) {
				return "AKIDNOPE", ""
			},
			wantErr: gorm.ErrRecordNotFound,
		},
		{
			name: "unknown scope user id is not found",
			setup: func(t *testing.T, d *DAO) (string, string) {
				u := mustCreateUser(t, d, "owner")
				k := mustCreateKey(t, d, u, "k")
				return k.AccessKeyID, "uuid-does-not-exist"
			},
			wantErr: gorm.ErrRecordNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			d := openTestDAO(t)
			keyID, userID := tt.setup(t, d)

			err := d.DisableKey(context.Background(), keyID, "test reason", userID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DisableKey err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestListKeys(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T, d *DAO) (userID string, wantCount int)
		wantErr error
	}{
		{
			name: "lists all keys when no user id",
			setup: func(t *testing.T, d *DAO) (string, int) {
				a := mustCreateUser(t, d, "a")
				mustCreateKey(t, d, a, "a1")
				mustCreateKey(t, d, a, "a2")
				b := mustCreateUser(t, d, "b")
				mustCreateKey(t, d, b, "b1")
				return "", 3
			},
		},
		{
			name: "scoped to a single user",
			setup: func(t *testing.T, d *DAO) (string, int) {
				a := mustCreateUser(t, d, "a")
				mustCreateKey(t, d, a, "a1")
				mustCreateKey(t, d, a, "a2")
				b := mustCreateUser(t, d, "b")
				mustCreateKey(t, d, b, "b1")
				return a.UUID, 2
			},
		},
		{
			name: "unknown user id is not found",
			setup: func(t *testing.T, d *DAO) (string, int) {
				return "uuid-does-not-exist", 0
			},
			wantErr: gorm.ErrRecordNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			d := openTestDAO(t)
			userID, wantCount := tt.setup(t, d)

			got, err := d.ListKeys(context.Background(), 100, 0, userID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ListKeys err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && len(got) != wantCount {
				t.Errorf("ListKeys count = %d, want %d", len(got), wantCount)
			}
		})
	}
}
