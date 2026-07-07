package authctx

import (
	"context"
	"testing"
	"time"

	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

func TestKeyID(t *testing.T) {
	tests := []struct {
		name   string
		ctx    func() context.Context
		wantID string
		wantOK bool
	}{
		{
			name:   "round trip",
			ctx:    func() context.Context { return WithKeyID(context.Background(), "AKIDEXAMPLE") },
			wantID: "AKIDEXAMPLE",
			wantOK: true,
		},
		{
			name:   "absent",
			ctx:    func() context.Context { return context.Background() },
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := KeyID(tt.ctx())
			if id != tt.wantID || ok != tt.wantOK {
				t.Errorf("KeyID() = (%q, %v), want (%q, %v)", id, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestUser(t *testing.T) {
	want := &iamv1.User{Id: "u-1"}

	tests := []struct {
		name   string
		ctx    func() context.Context
		wantOK bool
	}{
		{
			name:   "round trip",
			ctx:    func() context.Context { return WithUser(context.Background(), want) },
			wantOK: true,
		},
		{
			name:   "absent",
			ctx:    func() context.Context { return context.Background() },
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, ok := User(tt.ctx())
			if ok != tt.wantOK {
				t.Fatalf("User() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && u.GetId() != want.GetId() {
				t.Errorf("User() = %v, want %v", u, want)
			}
		})
	}
}

func TestCaller(t *testing.T) {
	want := &Identity{
		AccessKeyID:    "AKIDEXAMPLE",
		OrganizationID: "org-1",
		PrincipalID:    "principal-1",
		DisplayName:    "tester",
		SignedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name   string
		ctx    func() context.Context
		wantOK bool
	}{
		{
			name:   "round trip",
			ctx:    func() context.Context { return WithCaller(context.Background(), want) },
			wantOK: true,
		},
		{
			name:   "absent",
			ctx:    func() context.Context { return context.Background() },
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, ok := Caller(tt.ctx())
			if ok != tt.wantOK {
				t.Fatalf("Caller() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && *c != *want {
				t.Errorf("Caller() = %+v, want %+v", c, want)
			}
		})
	}
}
