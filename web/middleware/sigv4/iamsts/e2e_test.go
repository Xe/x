package iamsts

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/twitchtv/twirp"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/sigv4"
	"within.website/x/web/middleware/sigv4/sigv4client"
)

const (
	testKey    = "AKIDEXAMPLE"
	testSecret = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testRegion = "us-east-1"
	testSvc    = "execute-api"
)

// realSTS validates forwarded request material with a real sigv4.Verifier,
// mirroring cmd/iamd's STS service, so these tests exercise actual signature
// verification rather than a stubbed identity.
type realSTS struct {
	v *sigv4.Verifier
}

func (s realSTS) GetCallerIdentity(_ context.Context, req *stsv1.GetCallerIdentityReq) (*stsv1.GetCallerIdentityResp, error) {
	headers := make(http.Header, len(req.GetHeaders()))
	for _, h := range req.GetHeaders() {
		headers.Add(h.GetName(), h.GetValue())
	}
	declared := headers.Get("X-Amz-Content-Sha256")
	if declared == "" {
		return nil, twirp.InvalidArgumentError("x-amz-content-sha256", "must be forwarded so the signature can be verified without the body")
	}

	keyID, err := s.v.VerifySignature(req.GetMethod(), req.GetPath(), req.GetQuery(), req.GetHost(), headers, declared)
	if err != nil {
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}
	return &stsv1.GetCallerIdentityResp{
		AccessKeyId: keyID,
		User:        &iamv1.User{Id: "u1", Name: "tester"},
	}, nil
}

// TestEndToEnd_SigV4ClientToSTS drives the full central-validation chain:
// sigv4client signs an outgoing request, the iamsts middleware forwards the
// received material to an STS server, and that server verifies the signature
// with the sigv4 package. This is the deployment shape of cmd/iamd, so any
// canonicalization or header-forwarding disagreement between the three parts
// fails here.
func TestEndToEnd_SigV4ClientToSTS(t *testing.T) {
	verifier := &sigv4.Verifier{
		Region:  testRegion,
		Service: testSvc,
		Lookup: sigv4.LookuperFunc(func(id string) (string, error) {
			if id == testKey {
				return testSecret, nil
			}
			return "", sigv4.ErrUnknownKey
		}),
	}
	stsSrv := httptest.NewServer(stsv1.NewSTSServiceServer(realSTS{v: verifier}))
	defer stsSrv.Close()

	var gotKey, gotBody string
	v := NewVerifier(stsSrv.URL, http.DefaultClient, 1<<20)
	svc := httptest.NewServer(v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, _ := Caller(r.Context())
		gotKey = caller.AccessKeyID
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	})))
	defer svc.Close()

	rt, err := sigv4client.NewSigV4RoundTripper(&sigv4client.Config{
		Region:      testRegion,
		AccessKey:   testKey,
		SecretKey:   testSecret,
		ServiceName: testSvc,
	}, nil)
	if err != nil {
		t.Fatalf("new round tripper: %v", err)
	}
	client := &http.Client{Transport: rt}

	t.Run("GET", func(t *testing.T) {
		gotKey, gotBody = "", ""
		resp, err := client.Get(svc.URL + "/v1/things?list=1&list-type=2")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if gotKey != testKey {
			t.Fatalf("caller key = %q, want %q", gotKey, testKey)
		}
	})

	t.Run("POST", func(t *testing.T) {
		gotKey, gotBody = "", ""
		body := `{"hello":"world"}`
		resp, err := client.Post(svc.URL+"/v1/submit", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if gotKey != testKey {
			t.Fatalf("caller key = %q, want %q", gotKey, testKey)
		}
		if gotBody != body {
			t.Fatalf("body = %q, want %q", gotBody, body)
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		badRT, err := sigv4client.NewSigV4RoundTripper(&sigv4client.Config{
			Region:      testRegion,
			AccessKey:   "AKIDWRONG",
			SecretKey:   testSecret,
			ServiceName: testSvc,
		}, nil)
		if err != nil {
			t.Fatalf("new round tripper: %v", err)
		}
		resp, err := (&http.Client{Transport: badRT}).Get(svc.URL + "/v1/things")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})
}
