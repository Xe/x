package keys

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/twitchtv/twirp"
	"within.website/x/cmd/iamd/models"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/sigv4"
)

func newTestServer(t *testing.T) (*Server, *models.DAO) {
	t.Helper()
	d, err := models.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("models.Open: %v", err)
	}
	return New(d), d
}

func mustCreateUser(t *testing.T, d *models.DAO, name string) *models.User {
	t.Helper()
	u, err := d.CreateUser(context.Background(), name)
	if err != nil {
		t.Fatalf("CreateUser(%q): %v", name, err)
	}
	return u
}

// mustMakeAdmin flips is_admin for u in the store and in memory. CreateUser
// always provisions a non-admin user; the bootstrap path is the only thing that
// grants admin in production, so tests do it directly.
func mustMakeAdmin(t *testing.T, d *models.DAO, u *models.User) {
	t.Helper()
	if err := d.DB().Model(&models.User{}).Where("id = ?", u.Model.ID).Update("is_admin", true).Error; err != nil {
		t.Fatalf("make admin: %v", err)
	}
	u.IsAdmin = true
}

func mustCreateKey(t *testing.T, d *models.DAO, u *models.User) *models.Key {
	t.Helper()
	k, err := d.CreateKey(context.Background(), u, "test key")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	return k
}

// ctxWithCaller returns a context whose authenticated caller is u, mirroring
// what UserMiddleware stashes for a real request.
func ctxWithCaller(u *models.User) context.Context {
	return sigv4.WithUser(context.Background(), &iamv1.User{
		Id:      u.UUID,
		IsAdmin: u.IsAdmin,
	})
}

// twirpCode extracts the Twirp error code, defaulting to Internal for non-Twirp
// errors. A nil error has no code; callers gate success on err == nil.
func twirpCode(err error) twirp.ErrorCode {
	var tw twirp.Error
	if errors.As(err, &tw) {
		return tw.Code()
	}
	return twirp.Internal
}

func TestServer_CreateKey(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(t *testing.T, d *models.DAO) (ctx context.Context, userID string)
		comment  string
		wantOK   bool
		wantCode twirp.ErrorCode
	}{
		{
			name: "missing comment is invalid argument",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				return ctxWithCaller(a), mustCreateUser(t, d, "target").UUID
			},
			comment:  "",
			wantCode: twirp.InvalidArgument,
		},
		{
			name: "missing user id is invalid argument",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				return ctxWithCaller(a), ""
			},
			comment:  "c",
			wantCode: twirp.InvalidArgument,
		},
		{
			name: "non-admin creating for another user is permission denied",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				other := mustCreateUser(t, d, "other")
				target := mustCreateUser(t, d, "target")
				return ctxWithCaller(other), target.UUID
			},
			comment:  "c",
			wantCode: twirp.PermissionDenied,
		},
		{
			name: "non-admin creating for self succeeds",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				u := mustCreateUser(t, d, "self")
				return ctxWithCaller(u), u.UUID
			},
			comment: "my key",
			wantOK:  true,
		},
		{
			name: "admin creating for another user succeeds",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				target := mustCreateUser(t, d, "target")
				return ctxWithCaller(a), target.UUID
			},
			comment: "admin-issued",
			wantOK:  true,
		},
		{
			name: "unknown user id is not found",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				return ctxWithCaller(a), "uuid-nope"
			},
			comment:  "c",
			wantCode: twirp.NotFound,
		},
		{
			name: "no caller is unauthenticated",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				return context.Background(), mustCreateUser(t, d, "target").UUID
			},
			comment:  "c",
			wantCode: twirp.Unauthenticated,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, d := newTestServer(t)
			ctx, userID := tt.setup(t, d)

			resp, err := s.CreateKey(ctx, &iamv1.CreateKeyReq{
				Comment: tt.comment,
				UserId:  userID,
			})
			if tt.wantOK {
				if err != nil {
					t.Fatalf("CreateKey: %v", err)
				}
				if resp.GetKey().GetAccessKeyId() == "" {
					t.Error("empty access key id")
				}
				if resp.GetSecretAccessKey() == "" {
					t.Error("empty secret access key (should be returned once)")
				}
			} else if code := twirpCode(err); code != tt.wantCode {
				t.Fatalf("code = %s, want %s (err = %v)", code, tt.wantCode, err)
			}
		})
	}
}

func TestServer_DisableKey(t *testing.T) {
	cases := []struct {
		name     string
		reason   string
		setup    func(t *testing.T, d *models.DAO) (ctx context.Context, keyID string)
		wantOK   bool
		wantCode twirp.ErrorCode
	}{
		{
			name:   "owner disables own key",
			reason: "test",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				u := mustCreateUser(t, d, "owner")
				k := mustCreateKey(t, d, u)
				return ctxWithCaller(u), k.AccessKeyID
			},
			wantOK: true,
		},
		{
			name:   "non-admin disabling another's key is not found",
			reason: "test",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				owner := mustCreateUser(t, d, "owner")
				k := mustCreateKey(t, d, owner)
				other := mustCreateUser(t, d, "other")
				return ctxWithCaller(other), k.AccessKeyID
			},
			wantCode: twirp.NotFound,
		},
		{
			name:   "admin disables another user's key",
			reason: "test",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				owner := mustCreateUser(t, d, "owner")
				k := mustCreateKey(t, d, owner)
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				return ctxWithCaller(a), k.AccessKeyID
			},
			wantOK: true,
		},
		{
			name:   "unknown key is not found",
			reason: "test",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				return ctxWithCaller(a), "AKIDNOPE"
			},
			wantCode: twirp.NotFound,
		},
		{
			name:   "missing reason is invalid argument",
			reason: "",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				owner := mustCreateUser(t, d, "owner")
				k := mustCreateKey(t, d, owner)
				return ctxWithCaller(a), k.AccessKeyID
			},
			wantCode: twirp.InvalidArgument,
		},
		{
			name:   "no caller is unauthenticated",
			reason: "test",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string) {
				owner := mustCreateUser(t, d, "owner")
				k := mustCreateKey(t, d, owner)
				return context.Background(), k.AccessKeyID
			},
			wantCode: twirp.Unauthenticated,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, d := newTestServer(t)
			ctx, keyID := tt.setup(t, d)

			// user_id is deliberately unset: the caller's identity drives scope,
			// not the request field.
			_, err := s.DisableKey(ctx, &iamv1.DisableKeyReq{
				KeyId:  keyID,
				Reason: tt.reason,
			})
			if tt.wantOK {
				if err != nil {
					t.Fatalf("DisableKey: %v", err)
				}
			} else if code := twirpCode(err); code != tt.wantCode {
				t.Fatalf("code = %s, want %s (err = %v)", code, tt.wantCode, err)
			}
		})
	}
}

func TestServer_ListKeys(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(t *testing.T, d *models.DAO) (ctx context.Context, userID string, wantCount int)
		wantCode twirp.ErrorCode
	}{
		{
			name: "non-admin lists own keys",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string, int) {
				owner := mustCreateUser(t, d, "owner")
				mustCreateKey(t, d, owner)
				return ctxWithCaller(owner), "", 1
			},
		},
		{
			name: "non-admin requesting another's keys is permission denied",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string, int) {
				owner := mustCreateUser(t, d, "owner")
				mustCreateKey(t, d, owner)
				other := mustCreateUser(t, d, "other")
				return ctxWithCaller(other), owner.UUID, 0
			},
			wantCode: twirp.PermissionDenied,
		},
		{
			name: "admin lists keys across all users",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string, int) {
				owner := mustCreateUser(t, d, "owner")
				mustCreateKey(t, d, owner)
				other := mustCreateUser(t, d, "other")
				mustCreateKey(t, d, other)
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				return ctxWithCaller(a), "", 2
			},
		},
		{
			name: "admin scoped to a single user",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string, int) {
				owner := mustCreateUser(t, d, "owner")
				mustCreateKey(t, d, owner)
				other := mustCreateUser(t, d, "other")
				mustCreateKey(t, d, other)
				a := mustCreateUser(t, d, "admin")
				mustMakeAdmin(t, d, a)
				return ctxWithCaller(a), owner.UUID, 1
			},
		},
		{
			name: "no caller is unauthenticated",
			setup: func(t *testing.T, d *models.DAO) (context.Context, string, int) {
				owner := mustCreateUser(t, d, "owner")
				mustCreateKey(t, d, owner)
				return context.Background(), "", 0
			},
			wantCode: twirp.Unauthenticated,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, d := newTestServer(t)
			ctx, userID, wantCount := tt.setup(t, d)

			resp, err := s.ListKeys(ctx, &iamv1.ListKeysReq{
				Count:  100,
				Page:   0,
				UserId: userID,
			})
			if tt.wantCode != "" {
				if code := twirpCode(err); code != tt.wantCode {
					t.Fatalf("code = %s, want %s (err = %v)", code, tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListKeys: %v", err)
			}
			if len(resp.GetKeys()) != wantCount {
				t.Errorf("count = %d, want %d", len(resp.GetKeys()), wantCount)
			}
		})
	}
}
