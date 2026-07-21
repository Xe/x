package sigv4

import (
	"context"
	"errors"
	"testing"

	"github.com/twitchtv/twirp"
)

const validAuthHeader = "AWS4-HMAC-SHA256 " +
	"Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request, " +
	"SignedHeaders=host;x-amz-date, " +
	"Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7"

// Duplicate Authorization fields must be rejected so a future change cannot
// accidentally let an attacker overwrite an earlier, valid value.
func TestParseAuthHeaderRejectsDuplicateFields(t *testing.T) {
	cases := []struct {
		name string
		h    string
	}{
		{
			name: "duplicate Credential",
			h: "AWS4-HMAC-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request, " +
				"Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request, " +
				"SignedHeaders=host;x-amz-date, " +
				"Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7",
		},
		{
			name: "duplicate SignedHeaders",
			h: "AWS4-HMAC-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request, " +
				"SignedHeaders=host;x-amz-date, " +
				"SignedHeaders=host;x-amz-date, " +
				"Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7",
		},
		{
			name: "duplicate Signature",
			h: "AWS4-HMAC-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request, " +
				"SignedHeaders=host;x-amz-date, " +
				"Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7, " +
				"Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseCredential(tc.h); !errors.Is(err, ErrMalformedAuth) {
				t.Fatalf("err = %v, want ErrMalformedAuth", err)
			}
		})
	}
}

// A well-formed header with exactly one of each field must still parse.
func TestParseAuthHeaderAcceptsWellFormed(t *testing.T) {
	c, err := ParseCredential(validAuthHeader)
	if err != nil {
		t.Fatalf("ParseCredential: %v", err)
	}
	want := Credential{AccessKeyID: "AKIDEXAMPLE", Date: "20150830", Region: "us-east-1", Service: "iam"}
	if *c != want {
		t.Errorf("credential = %+v, want %+v", *c, want)
	}
}

// TestTwirpErrorMapping pins the twirp code each sentinel lands on. A
// malformed Authorization header is the caller's fault and maps to
// InvalidArgument (400); a missing header or an unknown/unsigned caller maps
// to an authentication-failure code so a probe cannot tell the cases apart.
func TestTwirpErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode twirp.ErrorCode
	}{
		{name: "missing auth", err: ErrMissingAuth, wantCode: twirp.Unauthenticated},
		{name: "malformed auth", err: ErrMalformedAuth, wantCode: twirp.InvalidArgument},
		{name: "unknown key", err: ErrUnknownKey, wantCode: twirp.PermissionDenied},
		{name: "unauthorized", err: ErrUnauthorized, wantCode: twirp.PermissionDenied},
		{name: "scope mismatch", err: ErrScopeMismatch, wantCode: twirp.InvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tErr := TwirpError(context.Background(), tc.err)
			tw, ok := tErr.(twirp.Error)
			if !ok {
				t.Fatalf("TwirpError returned non-twirp error: %v", tErr)
			}
			if tw.Code() != tc.wantCode {
				t.Fatalf("code = %q, want %q (msg=%q)", tw.Code(), tc.wantCode, tw.Msg())
			}
		})
	}
}
