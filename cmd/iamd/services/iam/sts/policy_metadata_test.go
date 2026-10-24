package sts

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"
	"within.website/x/cmd/iamd/models"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

func TestPolicyMetadata(t *testing.T) {
	for _, tt := range []struct {
		name  string
		admin bool
	}{
		{name: "policy"},
		{name: "admin", admin: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s, dao, akid, _ := newSigningKeysTest(t)
			key, err := dao.GetKeyWithUser(ctx, akid)
			if err != nil {
				t.Fatal(err)
			}
			p, err := dao.CreatePolicy(ctx, "downstream-access", []*iamv1.PolicyStatement{{
				Effect: iamv1.Effect_EFFECT_ALLOW, Methods: []string{"example.Things/Get"},
				Filter: "req.id == principal.principal_id",
			}})
			if err != nil {
				t.Fatal(err)
			}
			if err := dao.AttachPolicy(ctx, p.GetId(), key.User.UUID); err != nil {
				t.Fatal(err)
			}
			if tt.admin {
				if err := dao.DB().Model(&models.User{}).Where("uuid = ?", key.User.UUID).Update("is_admin", true).Error; err != nil {
					t.Fatal(err)
				}
				// Admin responses must not load irrelevant policy documents.
				if err := dao.DB().Model(&models.Policy{}).Where("uuid = ?", p.GetId()).Update("statements", "invalid JSON").Error; err != nil {
					t.Fatal(err)
				}
			}

			check := func(t *testing.T, id *stsv1.TokenIdentity, ps []*iamv1.Policy) {
				t.Helper()
				if id.GetPrincipalId() != key.User.UUID || id.GetIsAdmin() != tt.admin {
					t.Fatalf("identity = %v, want principal %s and admin %v", id, key.User.UUID, tt.admin)
				}
				if tt.admin {
					if len(ps) != 0 {
						t.Fatalf("admin policies = %v, want none", ps)
					}
				} else if len(ps) != 1 || !proto.Equal(ps[0], p) {
					t.Fatalf("policies = %v, want [%v]", ps, p)
				}
			}
			t.Run("SigV4", func(t *testing.T) {
				resp, err := s.GetSigningKey(ctx, validReq(akid))
				if err != nil {
					t.Fatal(err)
				}
				check(t, resp.GetIdentity(), resp.GetPolicies())
			})
			t.Run("SigV4A", func(t *testing.T) {
				resp, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: akid})
				if err != nil {
					t.Fatal(err)
				}
				check(t, resp.GetIdentity(), resp.GetPolicies())
			})
		})
	}
}
