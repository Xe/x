package sigv4a

import (
	"context"
	"testing"

	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/sigv4"
)

// TestCrossFamilyUser proves the point of authctx: a caller stored through
// one middleware family's WithUser is readable through the other family's
// User, because both delegate to the same authctx storage. This lives here
// (rather than in authctx_test.go, where it would read more naturally)
// because authctx must not import sigv4/sigv4a — doing so from authctx's own
// test files creates an import cycle (sigv4/sigv4a import authctx).
func TestCrossFamilyUser(t *testing.T) {
	want := &iamv1.User{Id: "cross-family-user"}
	ctx := WithUser(context.Background(), want)

	got, ok := sigv4.User(ctx)
	if !ok {
		t.Fatal("sigv4.User: ok = false, want true")
	}
	if got.GetId() != want.GetId() {
		t.Errorf("sigv4.User = %v, want %v", got, want)
	}
}
