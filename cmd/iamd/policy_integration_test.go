package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/twitchtv/twirp"
	"within.website/x/cmd/iamd/models"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/authctx"
	"within.website/x/web/middleware/sigv4/iamsts"
)

// TestIntegration_PolicyEnforcement drives the full authorization path over
// signed RPCs: an admin manages policies through PolicyService, a non-admin
// user is allowed exactly what its attached document grants, everything else
// is denied, and the admin bypass still works.
func TestIntegration_PolicyEnforcement(t *testing.T) {
	ctx := context.Background()
	dao := newDAO(t)
	adminAKID, adminSecret := bootstrapCreds(t, dao)

	eval, err := newAuthorizer()
	if err != nil {
		t.Fatalf("newAuthorizer: %v", err)
	}
	authMW := newDualVerifier(dao, intRegion, intService, 1<<20)
	srv := httptest.NewServer(newMux(quietLogger(), dao, authMW, eval, 5*time.Minute))
	defer srv.Close()

	hc := &http.Client{Transport: signedTransport(t, adminAKID, adminSecret)}
	usersAPI := iamv1.NewUserServiceProtobufClient(srv.URL, hc)
	policiesAPI := iamv1.NewPolicyServiceProtobufClient(srv.URL, hc)

	created, err := usersAPI.CreateUser(ctx, &iamv1.CreateUserReq{Name: "alice"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	aliceUUID := created.GetUser().GetId()
	aliceKeys := iamv1.NewKeyServiceProtobufClient(srv.URL, &http.Client{Transport: signedTransport(t, created.GetAccessKeyId(), created.GetSecretAccessKey())})
	aliceUsers := iamv1.NewUserServiceProtobufClient(srv.URL, &http.Client{Transport: signedTransport(t, created.GetAccessKeyId(), created.GetSecretAccessKey())})

	// Before any policy: alice is implicitly denied everything non-self.
	// (ListKeys for her own user_id is still handler-clamped to herself, but
	// policy evaluation runs first and must deny.)
	if _, err := aliceKeys.ListKeys(ctx, &iamv1.ListKeysReq{Count: 10, Page: 1, UserId: aliceUUID}); err == nil {
		t.Fatal("ListKeys before any policy: allowed, want implicit deny")
	} else if !isPermissionDenied(err) {
		t.Fatalf("ListKeys before any policy: %v, want PermissionDenied", err)
	}

	// Admin writes a policy: alice may manage only her own keys, and may
	// never mint a key for anyone else (explicit deny).
	created2, err := policiesAPI.CreatePolicy(ctx, &iamv1.CreatePolicyReq{
		Policy: &iamv1.Policy{
			Name: "own-keys-only",
			Statements: []*iamv1.PolicyStatement{
				{
					Effect: iamv1.Effect_EFFECT_DENY,
					Methods: []string{
						"within.website.x.iam.v1.KeyService/CreateKey",
					},
					Filter: `req.user_id != principal.principal_id`,
				},
				{
					Effect: iamv1.Effect_EFFECT_ALLOW,
					Methods: []string{
						"within.website.x.iam.v1.KeyService/*",
					},
					Filter: `req.user_id == "" || req.user_id == principal.principal_id`,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	policyID := created2.GetPolicy().GetId()

	if _, err := policiesAPI.AttachPolicy(ctx, &iamv1.AttachPolicyReq{PolicyId: policyID, UserId: aliceUUID}); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}

	t.Run("allowed call passes the policy filter", func(t *testing.T) {
		if _, err := aliceKeys.ListKeys(ctx, &iamv1.ListKeysReq{Count: 10, Page: 1, UserId: aliceUUID}); err != nil {
			t.Fatalf("ListKeys own: %v", err)
		}
	})

	t.Run("call outside the policy surface is implicitly denied", func(t *testing.T) {
		if _, err := aliceUsers.ListUsers(ctx, &iamv1.ListUsersReq{Count: 10, Page: 1}); err == nil {
			t.Fatal("ListUsers: allowed, want implicit deny")
		} else if !isPermissionDenied(err) {
			t.Fatalf("ListUsers: %v, want PermissionDenied", err)
		}
	})

	t.Run("explicit deny beats allow", func(t *testing.T) {
		other, err := usersAPI.CreateUser(ctx, &iamv1.CreateUserReq{Name: "bob"})
		if err != nil {
			t.Fatalf("CreateUser(bob): %v", err)
		}
		_, err = aliceKeys.CreateKey(ctx, &iamv1.CreateKeyReq{Comment: "for bob", UserId: other.GetUser().GetId()})
		if err == nil {
			t.Fatal("CreateKey for another user: allowed, want explicit deny")
		}
		if !isPermissionDenied(err) {
			t.Fatalf("CreateKey for another user: %v, want PermissionDenied", err)
		}
		// The denial message names the denying policy document.
		if msg := err.Error(); !strings.Contains(msg, "deny by policy") {
			t.Errorf("denial message = %q, want it to name the deny statement", msg)
		}
	})

	t.Run("policy administration is admin-only", func(t *testing.T) {
		grant, err := policiesAPI.CreatePolicy(ctx, &iamv1.CreatePolicyReq{Policy: &iamv1.Policy{
			Name: "attempt-policy-administration",
			Statements: []*iamv1.PolicyStatement{{
				Effect: iamv1.Effect_EFFECT_ALLOW, Methods: []string{"within.website.x.iam.v1.PolicyService/*"},
			}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := policiesAPI.AttachPolicy(ctx, &iamv1.AttachPolicyReq{PolicyId: grant.GetPolicy().GetId(), UserId: aliceUUID}); err != nil {
			t.Fatal(err)
		}
		alicePolicies := iamv1.NewPolicyServiceProtobufClient(srv.URL, &http.Client{Transport: signedTransport(t, created.GetAccessKeyId(), created.GetSecretAccessKey())})
		if _, err := alicePolicies.ListPolicies(ctx, &iamv1.ListPoliciesReq{Count: 10, Page: 1}); err == nil {
			t.Fatal("ListPolicies by non-admin: allowed, want deny")
		} else if !isPermissionDenied(err) {
			t.Fatalf("ListPolicies by non-admin: %v, want PermissionDenied", err)
		}
	})

	t.Run("admin bypasses policies", func(t *testing.T) {
		admin, err := dao.GetKeyWithUser(ctx, adminAKID)
		if err != nil {
			t.Fatal(err)
		}
		broken, err := dao.CreatePolicy(ctx, "broken-admin-policy", []*iamv1.PolicyStatement{{
			Effect: iamv1.Effect_EFFECT_DENY, Methods: []string{"*"},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if err := dao.AttachPolicy(ctx, broken.GetId(), admin.User.UUID); err != nil {
			t.Fatal(err)
		}
		if err := dao.DB().Model(&models.Policy{}).Where("uuid = ?", broken.GetId()).Update("statements", "invalid JSON").Error; err != nil {
			t.Fatal(err)
		}
		if _, err := usersAPI.ListUsers(ctx, &iamv1.ListUsersReq{Count: 10, Page: 1}); err != nil {
			t.Fatalf("admin ListUsers: %v", err)
		}
		// Admins must retain the management access needed to repair an
		// attachment, even when its document cannot be decoded.
		if _, err := policiesAPI.DetachPolicy(ctx, &iamv1.DetachPolicyReq{PolicyId: broken.GetId(), UserId: admin.User.UUID}); err != nil {
			t.Fatalf("admin DetachPolicy: %v", err)
		}
	})

	t.Run("detach removes the grant", func(t *testing.T) {
		if _, err := policiesAPI.DetachPolicy(ctx, &iamv1.DetachPolicyReq{PolicyId: policyID, UserId: aliceUUID}); err != nil {
			t.Fatalf("DetachPolicy: %v", err)
		}
		if _, err := aliceKeys.ListKeys(ctx, &iamv1.ListKeysReq{Count: 10, Page: 1, UserId: aliceUUID}); err == nil {
			t.Fatal("ListKeys after detach: allowed, want implicit deny")
		}
	})
}

// TestIntegration_STSPropagatesPolicies checks that the signing-key service
// hands downstream verifiers the caller's policy set and admin flag, and
// that the verifier middleware surfaces both in the request context.
func TestIntegration_STSPropagatesPolicies(t *testing.T) {
	ctx := context.Background()
	dao := newDAO(t)
	verifierAKID, verifierSecret := bootstrapCreds(t, dao)

	eval, err := newAuthorizer()
	if err != nil {
		t.Fatalf("newAuthorizer: %v", err)
	}
	authMW := newDualVerifier(dao, intRegion, intService, 1<<20)
	iamd := httptest.NewServer(newMux(quietLogger(), dao, authMW, eval, 5*time.Minute))
	defer iamd.Close()

	endUser, err := dao.CreateUser(ctx, "end-user")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	endKey, err := dao.CreateKey(ctx, endUser, "end user key")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	// Attach a recognizable policy document directly through the DAO; the
	// admin client path is covered by TestIntegration_PolicyEnforcement.
	policy, err := dao.CreatePolicy(ctx, "downstream-grant", []*iamv1.PolicyStatement{{
		Effect:  iamv1.Effect_EFFECT_ALLOW,
		Methods: []string{"example.v1.Things/*"},
	}})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if err := dao.AttachPolicy(ctx, policy.GetId(), endUser.UUID); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}

	verifier := iamsts.New(iamsts.Config{
		BaseURL:     iamd.URL,
		HTTPClient:  &http.Client{Transport: signedTransport(t, verifierAKID, verifierSecret)},
		Region:      intRegion,
		Service:     intService,
		MaxBodySize: 1 << 20,
	})

	var gotPolicies, gotAdmin = -1, true
	app := httptest.NewServer(verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ps, ok := authctx.PrincipalPolicies(r.Context()); ok {
			gotPolicies = len(ps)
		}
		if c, ok := iamsts.Caller(r.Context()); ok {
			gotAdmin = c.IsAdmin
		}
		w.WriteHeader(http.StatusNoContent)
	})))
	defer app.Close()

	endUserClient := &http.Client{Transport: classicSignedTransport(t, endKey.AccessKeyID, endKey.SecretAccessKey)}
	resp, err := endUserClient.Get(app.URL + "/thing")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if gotPolicies != 1 {
		t.Errorf("downstream saw %d policies, want 1", gotPolicies)
	}
	if gotAdmin {
		t.Error("end user reported as admin downstream")
	}
}

func isPermissionDenied(err error) bool {
	te, ok := err.(twirp.Error)
	return ok && te.Code() == twirp.PermissionDenied
}
