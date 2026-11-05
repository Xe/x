package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

// ErrPolicyAttached is returned by DeletePolicy while at least one user still
// has the policy attached; detach first so no enforcement snapshot can
// reference a deleted document.
var ErrPolicyAttached = errors.New("policy attached to at least one user")

// ErrDuplicatePolicyName is returned by CreatePolicy when the name is
// already taken.
var ErrDuplicatePolicyName = errors.New("policy name already in use")

// ErrAlreadyAttached is returned by AttachPolicy when the exact (user,
// policy) edge already exists.
var ErrAlreadyAttached = errors.New("policy already attached to user")

// Policy is a named authorization document. Statements are stored as a JSON
// array of protojson-encoded iamv1.PolicyStatement values — proto field
// names, not Go ones — so the wire format is stable across regenerations.
type Policy struct {
	gorm.Model // adds CreatedAt, UpdatedAt, DeletedAt

	UUID string `gorm:"uniqueIndex"` // uuidv7
	Name string `gorm:"uniqueIndex"` // human-readable, unique

	Statements string
}

// UserPolicy is the attachment edge between a user and a policy. Rows are
// hard-deleted on detach: a soft-deleted row would keep occupying the unique
// (user_id, policy_id) index and block re-attaching.
type UserPolicy struct {
	gorm.Model

	UserID   uint `gorm:"uniqueIndex:idx_user_policy"`
	PolicyID uint `gorm:"uniqueIndex:idx_user_policy"`
}

// statementsDoc is the JSON envelope persisted in Policy.Statements.
type statementsDoc struct {
	Statements []json.RawMessage `json:"statements"`
}

func encodeStatements(stmts []*iamv1.PolicyStatement) (string, error) {
	doc := statementsDoc{Statements: make([]json.RawMessage, 0, len(stmts))}
	for _, s := range stmts {
		b, err := protojson.Marshal(s)
		if err != nil {
			return "", fmt.Errorf("encode statement: %w", err)
		}
		doc.Statements = append(doc.Statements, b)
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("encode statements: %w", err)
	}
	return string(b), nil
}

func decodeStatements(s string) ([]*iamv1.PolicyStatement, error) {
	var doc statementsDoc
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		return nil, fmt.Errorf("decode statements: %w", err)
	}
	stmts := make([]*iamv1.PolicyStatement, 0, len(doc.Statements))
	for _, raw := range doc.Statements {
		st := &iamv1.PolicyStatement{}
		if err := protojson.Unmarshal(raw, st); err != nil {
			return nil, fmt.Errorf("decode statement: %w", err)
		}
		stmts = append(stmts, st)
	}
	return stmts, nil
}

// AsProto converts the stored row to its proto form. It errors on corrupted
// statement payloads rather than silently dropping statements: a policy that
// half-loads must not half-apply.
func (p *Policy) AsProto() (*iamv1.Policy, error) {
	stmts, err := decodeStatements(p.Statements)
	if err != nil {
		return nil, fmt.Errorf("policy %s (%s): %w", p.UUID, p.Name, err)
	}
	return &iamv1.Policy{
		Id:         p.UUID,
		Name:       p.Name,
		Statements: stmts,
		CreatedAt:  timestamppb.New(p.CreatedAt),
		UpdatedAt:  timestamppb.New(p.UpdatedAt),
	}, nil
}

func asProtos(ps []Policy) ([]*iamv1.Policy, error) {
	out := make([]*iamv1.Policy, 0, len(ps))
	for i := range ps {
		p, err := ps[i].AsProto()
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// isUniqueViolation reports whether err is a unique-index violation from the
// SQLite driver. gorm's TranslateError is off in this deployment, so the
// constraint surfaces as a driver error string.
func isUniqueViolation(err error) bool {
	for ; err != nil; err = errors.Unwrap(err) {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return true
		}
	}
	return false
}

// CreatePolicy stores a new document with a fresh uuidv7 id. A duplicate
// name returns an error for which isUniqueViolation is true.
func (d *DAO) CreatePolicy(ctx context.Context, name string, stmts []*iamv1.PolicyStatement) (*iamv1.Policy, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	encoded, err := encodeStatements(stmts)
	if err != nil {
		return nil, err
	}

	p := Policy{
		UUID:       id.String(),
		Name:       name,
		Statements: encoded,
	}
	if err := d.policies.Create(ctx, &p); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicatePolicyName
		}
		return nil, err
	}

	return p.AsProto()
}

// GetPolicy returns the policy with the given public id (UUID), as proto.
// It returns an error wrapping gorm.ErrRecordNotFound if no such policy
// exists (or it is soft-deleted).
func (d *DAO) GetPolicy(ctx context.Context, policyID string) (*iamv1.Policy, error) {
	p, err := d.policies.Where("uuid = ?", policyID).First(ctx)
	if err != nil {
		return nil, err
	}
	return p.AsProto()
}

// ListPolicies enumerates documents, newest first, paginated.
func (d *DAO) ListPolicies(ctx context.Context, count, page int) ([]*iamv1.Policy, error) {
	ps, err := d.policies.
		Order("created_at DESC").
		Limit(count).
		Offset(count * page).
		Find(ctx)
	if err != nil {
		return nil, err
	}
	return asProtos(ps)
}

// DeletePolicy soft-deletes the document with the given public id. It
// refuses with ErrPolicyAttached while any live attachment exists.
func (d *DAO) DeletePolicy(ctx context.Context, policyID string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		policies := gorm.G[Policy](tx)
		userPolicies := gorm.G[UserPolicy](tx)
		p, err := policies.Where("uuid = ?", policyID).First(ctx)
		if err != nil {
			return err
		}

		edges, err := userPolicies.Where("policy_id = ?", p.Model.ID).Find(ctx)
		if err != nil {
			return err
		}
		if len(edges) > 0 {
			return ErrPolicyAttached
		}

		if _, err := policies.Where("uuid = ?", policyID).Delete(ctx); err != nil {
			return err
		}
		return nil
	})
}

// AttachPolicy binds a policy to a user, both by public ids. It returns an
// error wrapping gorm.ErrRecordNotFound when either side is missing, and
// ErrAlreadyAttached when the edge exists.
func (d *DAO) AttachPolicy(ctx context.Context, policyID, userID string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		policies := gorm.G[Policy](tx)
		userPolicies := gorm.G[UserPolicy](tx)
		users := gorm.G[User](tx)
		p, err := policies.Where("uuid = ?", policyID).First(ctx)
		if err != nil {
			return err
		}
		u, err := users.Where("uuid = ?", userID).First(ctx)
		if err != nil {
			return err
		}

		existing, err := userPolicies.Where("user_id = ? AND policy_id = ?", u.Model.ID, p.Model.ID).Find(ctx)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return ErrAlreadyAttached
		}

		edge := UserPolicy{UserID: u.Model.ID, PolicyID: p.Model.ID}
		if err := userPolicies.Create(ctx, &edge); err != nil {
			// A concurrent attach raced us onto the unique index.
			if isUniqueViolation(err) {
				return ErrAlreadyAttached
			}
			return err
		}
		return nil
	})
}

// DetachPolicy removes the (policy, user) edge. The edge row is
// hard-deleted so re-attaching later is not blocked by the unique index.
// It returns an error wrapping gorm.ErrRecordNotFound when the edge does
// not exist.
func (d *DAO) DetachPolicy(ctx context.Context, policyID, userID string) error {
	p, err := d.policies.Where("uuid = ?", policyID).First(ctx)
	if err != nil {
		return err
	}
	u, err := d.users.Where("uuid = ?", userID).First(ctx)
	if err != nil {
		return err
	}

	edges := gorm.G[UserPolicy](d.db.Unscoped())
	n, err := edges.Where("user_id = ? AND policy_id = ?", u.Model.ID, p.Model.ID).Delete(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// PoliciesForUser returns every live policy attached to the user with the
// given public id, as protos ready for evaluation. Attachments to missing
// rows fail closed rather than silently omitting a possible DENY statement;
// the user resolves or the whole call fails with gorm.ErrRecordNotFound.
func (d *DAO) PoliciesForUser(ctx context.Context, userID string) ([]*iamv1.Policy, error) {
	u, err := d.users.Where("uuid = ?", userID).First(ctx)
	if err != nil {
		return nil, err
	}

	edges, err := d.userPolicies.Where("user_id = ?", u.Model.ID).Find(ctx)
	if err != nil {
		return nil, err
	}
	if len(edges) == 0 {
		return nil, nil
	}

	ids := make([]uint, 0, len(edges))
	for _, e := range edges {
		ids = append(ids, e.PolicyID)
	}

	ps, err := d.policies.Where("id IN ?", ids).Order("created_at DESC").Find(ctx)
	if err != nil {
		return nil, err
	}
	if len(ps) != len(edges) {
		return nil, fmt.Errorf("user %s has an attachment to an unavailable policy", userID)
	}
	return asProtos(ps)
}
