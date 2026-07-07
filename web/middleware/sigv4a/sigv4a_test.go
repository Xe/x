package sigv4a

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// parseVectorRequest builds an *http.Request from a vector's request.txt or
// header-signed-request.txt.
func parseVectorRequest(t *testing.T, raw string) (*http.Request, []byte) {
	t.Helper()
	r, err := http.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return r, body
}

// vectorVerifier returns a Verifier configured for a vector's scope and
// clock, resolving only that vector's credential.
func vectorVerifier(t *testing.T, vc vectorContext) *Verifier {
	t.Helper()
	return &Verifier{
		Region:  vc.Region,
		Service: vc.Service,
		Lookup: LookuperFunc(func(id string) (string, error) {
			if id == vc.Credentials.AccessKeyID {
				return vc.Credentials.SecretAccessKey, nil
			}
			return "", ErrUnknownKey
		}),
		Now: func() time.Time { return vc.signingTime(t).Add(5 * time.Second) },
	}
}

// TestVectors_Verify feeds each pre-signed request from the AWS test suite
// through the Verifier: an independent implementation's signature must
// verify against the key derived from the same credentials.
func TestVectors_Verify(t *testing.T) {
	for _, dir := range vectorDirs(t) {
		t.Run(dir, func(t *testing.T) {
			vc := loadVectorContext(t, dir)
			req, _ := parseVectorRequest(t, readVectorFile(t, dir, "header-signed-request.txt"))
			got, err := vectorVerifier(t, vc).Verify(req)
			if err != nil {
				t.Fatalf("verify: %v", err)
			}
			if got != vc.Credentials.AccessKeyID {
				t.Fatalf("key = %q, want %q", got, vc.Credentials.AccessKeyID)
			}
		})
	}
}

// TestVerifyRejections perturbs the get-vanilla vector to exercise each
// rejection path.
func TestVerifyRejections(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(v *Verifier)
		tamper  func(req *http.Request)
		wantErr error
	}{
		{
			name: "tampered signature",
			tamper: func(req *http.Request) {
				req.Header.Set("Authorization", flipLastDigit(req.Header.Get("Authorization")))
			},
			wantErr: ErrUnauthorized,
		},
		{
			name: "clock skew",
			mutate: func(v *Verifier) {
				v.Now = func() time.Time { return time.Date(2015, 8, 30, 14, 0, 0, 0, time.UTC) } // +~1.5h
			},
			wantErr: ErrClockSkew,
		},
		{
			name: "unknown key",
			mutate: func(v *Verifier) {
				v.Lookup = LookuperFunc(func(string) (string, error) { return "", ErrUnknownKey })
			},
			wantErr: ErrUnknownKey,
		},
		{
			// A signature minted for one service must not verify against
			// another, even though the math would otherwise check out.
			name:    "wrong service scope",
			mutate:  func(v *Verifier) { v.Service = "other-svc" },
			wantErr: ErrScopeMismatch,
		},
		{
			// The verifier's region must be covered by the signed
			// X-Amz-Region-Set.
			name:    "region not in region set",
			mutate:  func(v *Verifier) { v.Region = "eu-west-1" },
			wantErr: ErrScopeMismatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vc := loadVectorContext(t, "get-vanilla")
			req, _ := parseVectorRequest(t, readVectorFile(t, "get-vanilla", "header-signed-request.txt"))
			if tc.tamper != nil {
				tc.tamper(req)
			}
			v := vectorVerifier(t, vc)
			if tc.mutate != nil {
				tc.mutate(v)
			}
			if _, err := v.Verify(req); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func flipLastDigit(auth string) string {
	b := []byte(auth)
	last := len(b) - 1
	if b[last] == '0' {
		b[last] = '1'
	} else {
		b[last] = '0'
	}
	return string(b)
}

// host must be a signed header; otherwise a captured request can be replayed
// against any other host sharing the same region/service verifier.
func TestUnsignedHostRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	req.Header.Set("Authorization", "AWS4-ECDSA-P256-SHA256 "+
		"Credential=AKIDEXAMPLE/20150830/service/aws4_request, "+
		"SignedHeaders=x-amz-date, Signature=0000")
	vc := loadVectorContext(t, "get-vanilla")
	if _, err := vectorVerifier(t, vc).Verify(req); err != ErrMissingSignedHost {
		t.Fatalf("err = %v, want ErrMissingSignedHost", err)
	}
}

// x-amz-region-set must be a signed header: it feeds the scope decision, so
// an unsigned value would let a relay rewrite the audience of a signature.
func TestUnsignedRegionSetRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	req.Header.Set("Authorization", "AWS4-ECDSA-P256-SHA256 "+
		"Credential=AKIDEXAMPLE/20150830/service/aws4_request, "+
		"SignedHeaders=host;x-amz-date, Signature=0000")
	vc := loadVectorContext(t, "get-vanilla")
	if _, err := vectorVerifier(t, vc).Verify(req); err != ErrMissingRegionSet {
		t.Fatalf("err = %v, want ErrMissingRegionSet", err)
	}
}

// A Verifier without a Lookup must return an error, not panic.
func TestNilLookuper(t *testing.T) {
	v := &Verifier{Region: "us-east-1", Service: "service"}
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	if _, err := v.Verify(req); err != ErrNotConfigured {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// TestParseCredential extracts the scope tuple from an Authorization header.
func TestParseCredential(t *testing.T) {
	const h = "AWS4-ECDSA-P256-SHA256 Credential=AKIDEXAMPLE/20150830/iam/aws4_request, SignedHeaders=host;x-amz-date;x-amz-region-set, Signature=3045022100aaaa022100bbbb"
	c, err := ParseCredential(h)
	if err != nil {
		t.Fatalf("ParseCredential: %v", err)
	}
	want := Credential{AccessKeyID: "AKIDEXAMPLE", Date: "20150830", Service: "iam"}
	if *c != want {
		t.Errorf("credential = %+v, want %+v", *c, want)
	}
	if _, err := ParseCredential("Bearer nope"); !errors.Is(err, ErrMissingAuth) {
		t.Errorf("bad header err = %v, want ErrMissingAuth", err)
	}
	if _, err := ParseCredential("AWS4-ECDSA-P256-SHA256 Credential=AKID/20150830/us-east-1/iam/aws4_request, SignedHeaders=host, Signature=00"); !errors.Is(err, ErrMissingAuth) {
		t.Errorf("5-part (SigV4-style) scope err = %v, want ErrMissingAuth", err)
	}
}
