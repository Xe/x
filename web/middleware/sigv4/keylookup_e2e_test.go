package sigv4_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"

	"within.website/x/web/middleware/sigv4"
)

const (
	klTestKey    = "AKIDEXAMPLE"
	klTestSecret = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	klRegion     = "us-east-1"
	klService    = "execute-api"
	// SHA-256 of the empty string: the payload hash of a bodyless GET.
	klEmptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// signGET signs a bodyless GET for target at ts with the AWS SDK v4 signer,
// the reference implementation.
func signGET(t *testing.T, target string, ts time.Time) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("X-Amz-Content-Sha256", klEmptySHA)
	if err := signer.NewSigner().SignHTTP(context.Background(),
		aws.Credentials{AccessKeyID: klTestKey, SecretAccessKey: klTestSecret},
		req, klEmptySHA, klService, klRegion, ts); err != nil {
		t.Fatalf("SignHTTP: %v", err)
	}
	return req
}

// TestVerify_KeyLookup verifies an SDK-signed request using only a derived
// signing key, and confirms the lookuper receives the literal scope strings
// from the Credential= component.
func TestVerify_KeyLookup(t *testing.T) {
	var gotScope []string
	v := &sigv4.Verifier{
		Region:  klRegion,
		Service: klService,
		KeyLookup: sigv4.SigningKeyLookuperFunc(func(_ context.Context, akid, date, region, service string) ([]byte, error) {
			gotScope = []string{akid, date, region, service}
			if akid != klTestKey {
				return nil, sigv4.ErrUnknownKey
			}
			return sigv4.DeriveSigningKey(klTestSecret, date, region, service), nil
		}),
	}

	now := time.Now().UTC()
	req := signGET(t, "https://svc.example.com/things?a=1", now)

	keyID, err := v.Verify(req)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if keyID != klTestKey {
		t.Errorf("keyID = %q, want %q", keyID, klTestKey)
	}
	wantScope := []string{klTestKey, now.Format("20060102"), klRegion, klService}
	if strings.Join(gotScope, "/") != strings.Join(wantScope, "/") {
		t.Errorf("scope = %v, want %v", gotScope, wantScope)
	}

	// A tampered signature must fail with ErrUnauthorized.
	bad := signGET(t, "https://svc.example.com/things?a=1", now)
	auth := bad.Header.Get("Authorization")
	bad.Header.Set("Authorization", auth[:len(auth)-1]+flipHex(auth[len(auth)-1]))
	if _, err := v.Verify(bad); err == nil {
		t.Error("tampered signature verified")
	}
}

// flipHex returns a different valid hex digit so the signature stays
// well-formed but wrong.
func flipHex(b byte) string {
	if b == '0' {
		return "1"
	}
	return "0"
}

// TestVerify_NeitherLookupConfigured pins the misconfiguration guard.
func TestVerify_NeitherLookupConfigured(t *testing.T) {
	v := &sigv4.Verifier{Region: klRegion, Service: klService}
	req := signGET(t, "https://svc.example.com/", time.Now().UTC())
	if _, err := v.Verify(req); err == nil {
		t.Error("Verify with no lookuper succeeded")
	}
}
