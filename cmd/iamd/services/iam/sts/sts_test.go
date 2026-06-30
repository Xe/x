package sts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/twitchtv/twirp"

	"within.website/x/cmd/iamd/models"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

const (
	testRegion = "us-east-1"
	testSvc    = "iam"
)

func signingTime() time.Time { return time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC) }

// errorCode extracts the Twirp error code, or "" if err is not a Twirp error.
func errorCode(err error) twirp.ErrorCode {
	var twirpErr twirp.Error
	if errors.As(err, &twirpErr) {
		return twirpErr.Code()
	}
	return ""
}

// newTestServer stands up a DAO with one user + key and a verifier whose clock
// sits 5s after signingTime, returning the server and the active key/user.
func newTestServer(t *testing.T) (*Server, *models.Key, *models.User) {
	t.Helper()
	dao, err := models.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	user, err := dao.CreateUser(context.Background(), "tester")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	key, err := dao.CreateKey(context.Background(), user, "test")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	verifier := &sigv4.Verifier{
		Region:  testRegion,
		Service: testSvc,
		Now:     func() time.Time { return signingTime().Add(5 * time.Second) },
		Lookup: sigv4.LookuperFunc(func(id string) (string, error) {
			secret, err := dao.SecretFor(context.Background(), id)
			if err != nil {
				return "", sigv4.ErrUnknownKey
			}
			return secret, nil
		}),
	}
	return New(dao, verifier), key, user
}

// signMaterial signs req in place and returns the STS request material built
// from it, mirroring what iamsts forwards (lowercased header names plus an
// explicit content-length, since net/http strips it from the header map).
func signMaterial(t *testing.T, req *http.Request, body []byte, akid, secret string) *stsv1.GetCallerIdentityReq {
	t.Helper()
	sum := sha256.Sum256(body)
	declared := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", declared)
	creds := aws.Credentials{AccessKeyID: akid, SecretAccessKey: secret}
	if err := v4.NewSigner().SignHTTP(req.Context(), creds, req, declared, testSvc, testRegion, signingTime()); err != nil {
		t.Fatalf("sign: %v", err)
	}
	out := &stsv1.GetCallerIdentityReq{
		Method: req.Method,
		Path:   req.URL.EscapedPath(),
		Query:  req.URL.RawQuery,
		Host:   req.Host,
	}
	for name, vals := range req.Header {
		for _, v := range vals {
			out.Headers = append(out.Headers, &stsv1.Header{Name: strings.ToLower(name), Value: v})
		}
	}
	if req.ContentLength > 0 {
		out.Headers = append(out.Headers, &stsv1.Header{Name: "content-length", Value: strconv.FormatInt(req.ContentLength, 10)})
	}
	return out
}

func TestGetCallerIdentity(t *testing.T) {
	srv, key, user := newTestServer(t)

	t.Run("valid GET resolves owning user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/v1/things?a=1&b=2", nil)
		stsReq := signMaterial(t, req, nil, key.AccessKeyID, key.SecretAccessKey)

		resp, err := srv.GetCallerIdentity(context.Background(), stsReq)
		if err != nil {
			t.Fatalf("GetCallerIdentity: %v", err)
		}
		if resp.GetAccessKeyId() != key.AccessKeyID {
			t.Errorf("access_key_id = %q, want %q", resp.GetAccessKeyId(), key.AccessKeyID)
		}
		if resp.GetUser().GetId() != user.UUID {
			t.Errorf("user id = %q, want %q", resp.GetUser().GetId(), user.UUID)
		}
		if resp.GetSignedAt() == nil {
			t.Error("signed_at is nil, want the X-Amz-Date timestamp")
		}
	})

	t.Run("valid POST with body verifies without the body", func(t *testing.T) {
		body := []byte(`{"hello":"world"}`)
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/submit", strings.NewReader(string(body)))
		stsReq := signMaterial(t, req, body, key.AccessKeyID, key.SecretAccessKey)

		resp, err := srv.GetCallerIdentity(context.Background(), stsReq)
		if err != nil {
			t.Fatalf("GetCallerIdentity: %v", err)
		}
		if resp.GetAccessKeyId() != key.AccessKeyID {
			t.Errorf("access_key_id = %q, want %q", resp.GetAccessKeyId(), key.AccessKeyID)
		}
	})

	t.Run("wrong secret is unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
		stsReq := signMaterial(t, req, nil, key.AccessKeyID, "wrong-secret")

		_, err := srv.GetCallerIdentity(context.Background(), stsReq)
		if got := errorCode(err); got != twirp.Unauthenticated {
			t.Fatalf("code = %q, want %q (err=%v)", got, twirp.Unauthenticated, err)
		}
	})

	t.Run("unknown access key is unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
		stsReq := signMaterial(t, req, nil, "AKIDNOPE", "whatever")

		_, err := srv.GetCallerIdentity(context.Background(), stsReq)
		if got := errorCode(err); got != twirp.Unauthenticated {
			t.Fatalf("code = %q, want %q (err=%v)", got, twirp.Unauthenticated, err)
		}
	})

	t.Run("missing method is invalid argument", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
		stsReq := signMaterial(t, req, nil, key.AccessKeyID, key.SecretAccessKey)
		stsReq.Method = ""

		_, err := srv.GetCallerIdentity(context.Background(), stsReq)
		if got := errorCode(err); got != twirp.InvalidArgument {
			t.Fatalf("code = %q, want %q (err=%v)", got, twirp.InvalidArgument, err)
		}
	})

	t.Run("missing host is invalid argument", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
		stsReq := signMaterial(t, req, nil, key.AccessKeyID, key.SecretAccessKey)
		stsReq.Host = ""

		_, err := srv.GetCallerIdentity(context.Background(), stsReq)
		if got := errorCode(err); got != twirp.InvalidArgument {
			t.Fatalf("code = %q, want %q (err=%v)", got, twirp.InvalidArgument, err)
		}
	})

	t.Run("missing content-sha256 is invalid argument", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
		stsReq := signMaterial(t, req, nil, key.AccessKeyID, key.SecretAccessKey)
		filtered := stsReq.Headers[:0]
		for _, h := range stsReq.Headers {
			if h.GetName() != "x-amz-content-sha256" {
				filtered = append(filtered, h)
			}
		}
		stsReq.Headers = filtered

		_, err := srv.GetCallerIdentity(context.Background(), stsReq)
		if got := errorCode(err); got != twirp.InvalidArgument {
			t.Fatalf("code = %q, want %q (err=%v)", got, twirp.InvalidArgument, err)
		}
	})
}
