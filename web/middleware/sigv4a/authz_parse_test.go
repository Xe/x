package sigv4a

import (
	"errors"
	"testing"
)

const validAuthHeader = "AWS4-ECDSA-P256-SHA256 " +
	"Credential=AKIDEXAMPLE/20150830/iam/aws4_request, " +
	"SignedHeaders=host;x-amz-date;x-amz-region-set, " +
	"Signature=3045022100aaaa022100bbbb"

// Duplicate Authorization fields must be rejected so a future change cannot
// accidentally let an attacker overwrite an earlier, valid value.
func TestParseAuthHeaderRejectsDuplicateFields(t *testing.T) {
	cases := []struct {
		name string
		h    string
	}{
		{
			name: "duplicate Credential",
			h: "AWS4-ECDSA-P256-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/iam/aws4_request, " +
				"Credential=AKIDEXAMPLE/20150830/iam/aws4_request, " +
				"SignedHeaders=host;x-amz-date;x-amz-region-set, " +
				"Signature=3045022100aaaa022100bbbb",
		},
		{
			name: "duplicate SignedHeaders",
			h: "AWS4-ECDSA-P256-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/iam/aws4_request, " +
				"SignedHeaders=host;x-amz-date;x-amz-region-set, " +
				"SignedHeaders=host;x-amz-date;x-amz-region-set, " +
				"Signature=3045022100aaaa022100bbbb",
		},
		{
			name: "duplicate Signature",
			h: "AWS4-ECDSA-P256-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/iam/aws4_request, " +
				"SignedHeaders=host;x-amz-date;x-amz-region-set, " +
				"Signature=3045022100aaaa022100bbbb, " +
				"Signature=3045022100aaaa022100bbbb",
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
	want := Credential{AccessKeyID: "AKIDEXAMPLE", Date: "20150830", Service: "iam"}
	if *c != want {
		t.Errorf("credential = %+v, want %+v", *c, want)
	}
}

// A header that carries the SigV4A prefix but is structurally invalid must
// report ErrMalformedAuth, not ErrMissingAuth: the auth header was present,
// it just could not be parsed.
func TestParseAuthHeaderMalformed(t *testing.T) {
	cases := []struct {
		name string
		h    string
	}{
		{
			name: "empty key=value pair",
			h:    "AWS4-ECDSA-P256-SHA256 Credential=, SignedHeaders=host, Signature=00",
		},
		{
			name: "bare token without equals",
			h: "AWS4-ECDSA-P256-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/iam/aws4_request, " +
				"SignedHeaders=host, bogus, Signature=00",
		},
		{
			name: "unknown authorization key",
			h: "AWS4-ECDSA-P256-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/iam/aws4_request, " +
				"SignedHeaders=host, FunFact=surprise, Signature=00",
		},
		{
			name: "missing Signature",
			h: "AWS4-ECDSA-P256-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/iam/aws4_request, " +
				"SignedHeaders=host",
		},
		{
			name: "credential scope missing terminator",
			h: "AWS4-ECDSA-P256-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/iam, " +
				"SignedHeaders=host, Signature=00",
		},
		{
			name: "credential scope has too many slashes",
			h: "AWS4-ECDSA-P256-SHA256 " +
				"Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request, " +
				"SignedHeaders=host, Signature=00",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCredential(tc.h)
			if !errors.Is(err, ErrMalformedAuth) {
				t.Fatalf("err = %v, want ErrMalformedAuth", err)
			}
			if errors.Is(err, ErrMissingAuth) {
				t.Fatalf("err = %v, must not be ErrMissingAuth", err)
			}
		})
	}
}

// A header without the SigV4A algorithm prefix is not a SigV4A header at all,
// so it stays on ErrMissingAuth (distinguished from the structurally-invalid
// cases above).
func TestParseAuthHeaderMissingOrForeign(t *testing.T) {
	cases := []struct {
		name string
		h    string
	}{
		{name: "empty", h: ""},
		{name: "bearer", h: "Bearer nope"},
		{name: "sigv4 algorithm", h: "AWS4-HMAC-SHA256 Credential=AKID/20150830/us-east-1/iam/aws4_request, SignedHeaders=host, Signature=00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCredential(tc.h)
			if !errors.Is(err, ErrMissingAuth) {
				t.Fatalf("err = %v, want ErrMissingAuth", err)
			}
			if errors.Is(err, ErrMalformedAuth) {
				t.Fatalf("err = %v, must not be ErrMalformedAuth", err)
			}
		})
	}
}
