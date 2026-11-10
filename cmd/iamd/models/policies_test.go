package models

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

func allowPolicyStmt(pattern string) []*iamv1.PolicyStatement {
	return []*iamv1.PolicyStatement{{
		Effect:  iamv1.Effect_EFFECT_ALLOW,
		Methods: []string{pattern},
		Filter:  `req.user_id == principal.principal_id`,
	}}
}

func mustCreatePolicy(t *testing.T, d *DAO, name string, stmts []*iamv1.PolicyStatement) *iamv1.Policy {
	t.Helper()
	p, err := d.CreatePolicy(context.Background(), name, stmts)
	if err != nil {
		t.Fatalf("CreatePolicy(%q): %v", name, err)
	}
	return p
}

func TestCreatePolicyRoundTrip(t *testing.T) {
	d := openTestDAO(t)

	p := mustCreatePolicy(t, d, "own-keys", allowPolicyStmt("within.website.x.iam.v1.KeyService/*"))

	if p.GetId() == "" {
		t.Error("CreatePolicy left id empty")
	}
	if len(p.GetStatements()) != 1 {
		t.Fatalf("stored statements = %d, want 1", len(p.GetStatements()))
	}
	st := p.GetStatements()[0]
	if st.GetEffect() != iamv1.Effect_EFFECT_ALLOW {
		t.Errorf("effect = %v, want ALLOW", st.GetEffect())
	}
	if got := st.GetMethods(); len(got) != 1 || got[0] != "within.website.x.iam.v1.KeyService/*" {
		t.Errorf("methods = %v, want [within.website.x.iam.v1.KeyService/*]", got)
	}
	if st.GetFilter() != `req.user_id == principal.principal_id` {
		t.Errorf("filter = %q, changed in storage", st.GetFilter())
	}

	got, err := d.GetPolicy(context.Background(), p.GetId())
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if got.GetName() != "own-keys" {
		t.Errorf("name = %q, want own-keys", got.GetName())
	}
	if len(got.GetStatements()) != 1 {
		t.Errorf("round-trip statements = %d, want 1", len(got.GetStatements()))
	}
}

func TestCreatePolicyDuplicateName(t *testing.T) {
	d := openTestDAO(t)

	mustCreatePolicy(t, d, "dup", allowPolicyStmt("*"))

	_, err := d.CreatePolicy(context.Background(), "dup", allowPolicyStmt("*"))
	if !errors.Is(err, ErrDuplicatePolicyName) {
		t.Errorf("duplicate name error = %v, want ErrDuplicatePolicyName", err)
	}
}

func TestGetPolicyMissing(t *testing.T) {
	d := openTestDAO(t)

	_, err := d.GetPolicy(context.Background(), "no-such-id")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("GetPolicy(missing) error = %v, want ErrRecordNotFound", err)
	}
}

func TestListPolicies(t *testing.T) {
	d := openTestDAO(t)

	mustCreatePolicy(t, d, "a", allowPolicyStmt("*"))
	mustCreatePolicy(t, d, "b", allowPolicyStmt("*"))
	mustCreatePolicy(t, d, "c", allowPolicyStmt("*"))

	got, err := d.ListPolicies(context.Background(), 2, 0)
	if err != nil {
		t.Fatalf("ListPolicies: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("page 0 count = %d, want 2", len(got))
	}
	if got[0].GetName() != "c" || got[1].GetName() != "b" {
		t.Errorf("page 0 = [%s %s], want newest first [c b]", got[0].GetName(), got[1].GetName())
	}

	got, err = d.ListPolicies(context.Background(), 2, 1)
	if err != nil {
		t.Fatalf("ListPolicies page 1: %v", err)
	}
	if len(got) != 1 || got[0].GetName() != "a" {
		t.Errorf("page 1 = %v, want [a]", got)
	}
}

func TestDeletePolicyAttachmentGuard(t *testing.T) {
	d := openTestDAO(t)
	ctx := context.Background()

	p := mustCreatePolicy(t, d, "guarded", allowPolicyStmt("*"))
	u := mustCreateUser(t, d, "alice")
	if err := d.AttachPolicy(ctx, p.GetId(), u.UUID); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}

	err := d.DeletePolicy(ctx, p.GetId())
	if !errors.Is(err, ErrPolicyAttached) {
		t.Fatalf("DeletePolicy while attached error = %v, want ErrPolicyAttached", err)
	}

	if err := d.DetachPolicy(ctx, p.GetId(), u.UUID); err != nil {
		t.Fatalf("DetachPolicy: %v", err)
	}
	if err := d.DeletePolicy(ctx, p.GetId()); err != nil {
		t.Fatalf("DeletePolicy after detach: %v", err)
	}
	if _, err := d.GetPolicy(ctx, p.GetId()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("GetPolicy after delete error = %v, want ErrRecordNotFound", err)
	}
}

func TestAttachDetachLifecycle(t *testing.T) {
	d := openTestDAO(t)
	ctx := context.Background()

	p := mustCreatePolicy(t, d, "lifecycle", allowPolicyStmt("*"))
	u := mustCreateUser(t, d, "bob")

	ps, err := d.PoliciesForUser(ctx, u.UUID)
	if err != nil {
		t.Fatalf("PoliciesForUser empty: %v", err)
	}
	if len(ps) != 0 {
		t.Fatalf("PoliciesForUser before attach = %d, want 0", len(ps))
	}

	if err := d.AttachPolicy(ctx, p.GetId(), u.UUID); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}

	err = d.AttachPolicy(ctx, p.GetId(), u.UUID)
	if !errors.Is(err, ErrAlreadyAttached) {
		t.Errorf("second attach error = %v, want ErrAlreadyAttached", err)
	}

	ps, err = d.PoliciesForUser(ctx, u.UUID)
	if err != nil {
		t.Fatalf("PoliciesForUser: %v", err)
	}
	if len(ps) != 1 || ps[0].GetName() != "lifecycle" {
		t.Fatalf("PoliciesForUser = %v, want [lifecycle]", ps)
	}

	// Detach is hard-delete: re-attach must succeed on the same edge.
	if err := d.DetachPolicy(ctx, p.GetId(), u.UUID); err != nil {
		t.Fatalf("DetachPolicy: %v", err)
	}
	if err := d.AttachPolicy(ctx, p.GetId(), u.UUID); err != nil {
		t.Fatalf("re-attach after detach: %v", err)
	}

	err = d.DetachPolicy(ctx, p.GetId(), u.UUID)
	if err != nil {
		t.Fatalf("final detach: %v", err)
	}
	err = d.DetachPolicy(ctx, p.GetId(), u.UUID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("detach missing edge error = %v, want ErrRecordNotFound", err)
	}
}

func TestAttachPolicyMissingSides(t *testing.T) {
	d := openTestDAO(t)
	ctx := context.Background()

	u := mustCreateUser(t, d, "carol")

	if err := d.AttachPolicy(ctx, "no-such-policy", u.UUID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("attach with missing policy error = %v, want ErrRecordNotFound", err)
	}

	p := mustCreatePolicy(t, d, "orphan", allowPolicyStmt("*"))
	if err := d.AttachPolicy(ctx, p.GetId(), "no-such-user"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("attach with missing user error = %v, want ErrRecordNotFound", err)
	}
}

func TestPoliciesForUserDisabledUser(t *testing.T) {
	d := openTestDAO(t)
	ctx := context.Background()

	u := mustCreateUser(t, d, "dave")
	p := mustCreatePolicy(t, d, "attached-then-disabled", allowPolicyStmt("*"))
	if err := d.AttachPolicy(ctx, p.GetId(), u.UUID); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}

	if err := d.DisableUser(ctx, u.UUID, "test done"); err != nil {
		t.Fatalf("DisableUser: %v", err)
	}

	// The user no longer resolves, so their policy set is unreachable —
	// authentication fails before authorization is ever consulted.
	if _, err := d.PoliciesForUser(ctx, u.UUID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("PoliciesForUser(disabled) error = %v, want ErrRecordNotFound", err)
	}
}

func TestPoliciesForUserRejectsUnavailableDocuments(t *testing.T) {
	for _, tt := range []struct {
		name        string
		breakPolicy func(*gorm.DB, string) error
	}{
		{name: "deleted", breakPolicy: func(db *gorm.DB, id string) error {
			return db.Where("uuid = ?", id).Delete(&Policy{}).Error
		}},
		{name: "missing", breakPolicy: func(db *gorm.DB, id string) error {
			return db.Unscoped().Where("uuid = ?", id).Delete(&Policy{}).Error
		}},
		{name: "corrupt", breakPolicy: func(db *gorm.DB, id string) error {
			return db.Model(&Policy{}).Where("uuid = ?", id).Update("statements", "invalid JSON").Error
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := openTestDAO(t)
			ctx := context.Background()
			u := mustCreateUser(t, d, "alice")
			allow := mustCreatePolicy(t, d, "allow", allowPolicyStmt("*"))
			deny := mustCreatePolicy(t, d, "deny", []*iamv1.PolicyStatement{{
				Effect: iamv1.Effect_EFFECT_DENY, Methods: []string{"*"},
			}})
			for _, p := range []*iamv1.Policy{allow, deny} {
				if err := d.AttachPolicy(ctx, p.GetId(), u.UUID); err != nil {
					t.Fatal(err)
				}
			}
			if err := tt.breakPolicy(d.db, deny.GetId()); err != nil {
				t.Fatal(err)
			}
			if ps, err := d.PoliciesForUser(ctx, u.UUID); err == nil {
				t.Fatalf("PoliciesForUser returned %v without its deny policy", ps)
			}
		})
	}
}
