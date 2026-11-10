// Package authctx holds the canonical context keys for the verified caller,
// shared by the sigv4 and sigv4a middleware families (and their iamsts
// sub-packages) so identity stored by either family is readable through
// either — and directly through this package, which new code should prefer.
package authctx

import (
	"context"
	"time"

	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

type keyIDKey struct{}

// WithKeyID stores the verified access key id in ctx.
func WithKeyID(ctx context.Context, keyID string) context.Context {
	return context.WithValue(ctx, keyIDKey{}, keyID)
}

// KeyID returns the verified access key id stored in ctx, if any.
func KeyID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(keyIDKey{}).(string)
	return id, ok
}

type userKey struct{}

// WithUser stores the IAM user resolved for the verified caller in ctx. It is
// the local-verification counterpart to WithCaller: middleware resolves the
// access key id from KeyID to its owning user and stashes it here so
// downstream code (logging interceptors, handlers) can attribute the call
// without re-querying the backing store.
func WithUser(ctx context.Context, u *iamv1.User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

// User returns the IAM user associated with the verified caller, if any. The
// bool is false when nothing populated it.
func User(ctx context.Context) (*iamv1.User, bool) {
	u, ok := ctx.Value(userKey{}).(*iamv1.User)
	return u, ok
}

// Identity is the verified caller stored in the request context on success.
// The fields come from the SigningKeyService response identity; SignedAt is
// parsed from the request's X-Amz-Date.
type Identity struct {
	AccessKeyID    string
	OrganizationID string
	PrincipalID    string
	DisplayName    string
	// IsAdmin propagates iamd's admin flag: admin callers bypass policy
	// evaluation everywhere, exactly as they do inside iamd.
	IsAdmin  bool
	SignedAt time.Time
}

type principalPoliciesKey struct{}

// WithPrincipalPolicies stores the caller's effective policy set in ctx. It
// is populated by the iamsts verifiers from the SigningKeyService response
// alongside WithCaller, so downstream services can authorize calls locally
// (see within.website/x/iam/authz) without a per-request IAM RPC.
func WithPrincipalPolicies(ctx context.Context, policies []*iamv1.Policy) context.Context {
	return context.WithValue(ctx, principalPoliciesKey{}, policies)
}

// PrincipalPolicies returns the caller's policy set stored in ctx, if any.
// The bool is false when nothing populated it (for example an admin caller,
// which carries no policies because it bypasses evaluation).
func PrincipalPolicies(ctx context.Context) ([]*iamv1.Policy, bool) {
	p, ok := ctx.Value(principalPoliciesKey{}).([]*iamv1.Policy)
	return p, ok
}

type callerKey struct{}

// WithCaller stores the verified iamsts caller identity in ctx.
func WithCaller(ctx context.Context, c *Identity) context.Context {
	return context.WithValue(ctx, callerKey{}, c)
}

// Caller returns the verified identity stored in ctx, if any.
func Caller(ctx context.Context) (*Identity, bool) {
	c, ok := ctx.Value(callerKey{}).(*Identity)
	return c, ok
}
