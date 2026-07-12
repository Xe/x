package iamsts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

const (
	testKey    = "AKIDEXAMPLE"
	testSecret = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testRegion = "us-east-1"
	testSvc    = "execute-api"
	emptySHA   = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// fakeKeys is a SigningKeyService stub served over a real Twirp endpoint. It
// derives real keys for testKey/testSecret, counts calls, and can be flipped
// to refuse.
type fakeKeys struct {
	mu         sync.Mutex
	calls      atomic.Int64
	refuseCode twirp.ErrorCode // when non-empty, refuse with this code
	cacheTTL   time.Duration
	now        func() time.Time
}

func (f *fakeKeys) setRefuse(code twirp.ErrorCode) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refuseCode = code
}

func (f *fakeKeys) GetPublicKey(ctx context.Context, req *stsv1.GetPublicKeyRequest) (*stsv1.GetPublicKeyResponse, error) {
	return nil, twirp.NewError(twirp.Unimplemented, "not implemented in this fake")
}

func (f *fakeKeys) GetSigningKey(_ context.Context, req *stsv1.GetSigningKeyRequest) (*stsv1.GetSigningKeyResponse, error) {
	f.calls.Add(1)
	f.mu.Lock()
	code := f.refuseCode
	f.mu.Unlock()
	if code != "" {
		return nil, twirp.NewError(code, "refused")
	}
	if req.GetAccessKeyId() != testKey {
		return nil, twirp.NotFoundError("unknown access key id")
	}
	now := f.now()
	day, err := time.Parse("20060102", req.GetDate())
	if err != nil {
		return nil, twirp.InvalidArgumentError("date", "bad date")
	}
	nva := day.AddDate(0, 0, 1).Add(15 * time.Minute)
	cu := now.Add(f.cacheTTL)
	if cu.After(nva) {
		cu = nva
	}
	return &stsv1.GetSigningKeyResponse{
		SigningKey: sigv4.DeriveSigningKey(testSecret, req.GetDate(), req.GetRegion(), req.GetService()),
		Identity: &stsv1.TokenIdentity{
			AccessKeyId: req.GetAccessKeyId(),
			PrincipalId: "u1",
			DisplayName: "tester",
		},
		NotValidAfter: timestamppb.New(nva),
		CacheUntil:    timestamppb.New(cu),
	}, nil
}

// harness wires a fakeKeys stub, a clock, and a Verifier-wrapped test handler.
type harness struct {
	fake     *fakeKeys
	verifier *Verifier
	handler  http.Handler
	now      time.Time
	mu       sync.Mutex
}

func (h *harness) clock() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.now
}

func (h *harness) advance(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.now = h.now.Add(d)
}

func newHarness(t *testing.T, cacheTTL time.Duration) (*harness, func()) {
	t.Helper()
	h := &harness{now: time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)}
	h.fake = &fakeKeys{cacheTTL: cacheTTL, now: h.clock}
	srv := httptest.NewServer(stsv1.NewSigningKeyServiceServer(h.fake))
	h.verifier = New(Config{
		BaseURL:     srv.URL,
		HTTPClient:  http.DefaultClient,
		Region:      testRegion,
		Service:     testSvc,
		MaxBodySize: 1 << 20,
		Now:         h.clock,
	})
	h.handler = h.verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	return h, srv.Close
}

// signedGET returns a bodyless GET signed at ts by the AWS SDK signer.
func signedGET(t *testing.T, ts time.Time) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "https://svc.example.com/things?a=1", nil)
	req.Header.Set("X-Amz-Content-Sha256", emptySHA)
	if err := signer.NewSigner().SignHTTP(context.Background(),
		aws.Credentials{AccessKeyID: testKey, SecretAccessKey: testSecret},
		req, emptySHA, testSvc, testRegion, ts); err != nil {
		t.Fatalf("SignHTTP: %v", err)
	}
	return req
}

func do(h *harness, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func TestVerifier_CacheHit(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("first request: status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := h.fake.calls.Load(); got != 1 {
		t.Fatalf("after miss: %d RPCs, want 1", got)
	}
	// Warm path: zero additional RPCs.
	for range 5 {
		if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
			t.Fatalf("warm request: status = %d", rec.Code)
		}
	}
	if got := h.fake.calls.Load(); got != 1 {
		t.Errorf("after warm requests: %d RPCs, want 1 (hot path must not call IAM)", got)
	}
}

func TestVerifier_IdentityThreaded(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	var got *Identity
	h.handler = h.verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = Caller(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if got == nil || got.AccessKeyID != testKey || got.PrincipalID != "u1" || got.DisplayName != "tester" {
		t.Fatalf("caller = %+v, want key %s principal u1", got, testKey)
	}
	if got.SignedAt.IsZero() {
		t.Error("SignedAt not populated")
	}
}

func TestVerifier_MidnightRollover(t *testing.T) {
	h, closeSrv := newHarness(t, time.Hour)
	defer closeSrv()

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("day-1 request: status = %d", rec.Code)
	}

	// Cross UTC midnight: the scope date changes, so the old key cannot
	// serve the new day — a fresh fetch must happen and the old slot must be
	// swept.
	h.advance(13 * time.Hour) // 12:00 -> 01:00 next day
	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("day-2 request: status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := h.fake.calls.Load(); got != 2 {
		t.Errorf("RPCs = %d, want 2 (one per scope date)", got)
	}
	if got := h.verifier.cacheLen(); got != 1 {
		t.Errorf("cache slots = %d, want 1 (day-1 key evicted)", got)
	}
}

func TestVerifier_RevocationAfterCacheUntil(t *testing.T) {
	h, closeSrv := newHarness(t, time.Minute)
	defer closeSrv()

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("initial request: status = %d", rec.Code)
	}

	// Key gets disabled server-side. Within cache_until the old key still
	// verifies (bounded staleness)...
	h.fake.setRefuse(twirp.PermissionDenied)
	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("within TTL: status = %d, want 204 (cached)", rec.Code)
	}

	// ...and after cache_until expires, the refetch sees the refusal and the
	// request fails like an unknown key.
	h.advance(2 * time.Minute)
	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusForbidden {
		t.Fatalf("after TTL: status = %d, want 403", rec.Code)
	}
	if got := h.fake.calls.Load(); got != 2 {
		t.Errorf("RPCs = %d, want 2 (refetch after cache_until)", got)
	}
}

func TestVerifier_NegativeCache(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()
	h.fake.setRefuse(twirp.NotFound)

	for range 5 {
		if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	}
	if got := h.fake.calls.Load(); got != 1 {
		t.Errorf("RPCs = %d, want 1 (refusal cached ~30s)", got)
	}

	// The negative entry expires and IAM is asked again.
	h.advance(time.Minute)
	do(h, signedGET(t, h.clock()))
	if got := h.fake.calls.Load(); got != 2 {
		t.Errorf("RPCs = %d, want 2 after negative TTL", got)
	}
}

func TestVerifier_Singleflight(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	const n = 8
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			do(h, signedGET(t, h.clock()))
		}()
	}
	wg.Wait()
	// All n concurrent misses for one scope must collapse into one RPC.
	if got := h.fake.calls.Load(); got != 1 {
		t.Errorf("RPCs = %d, want 1 (singleflight)", got)
	}
}

func TestVerifier_SignatureMismatch(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	req := signedGET(t, h.clock())
	req.URL.RawQuery = "a=2" // change what was signed
	if rec := do(h, req); rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (SignatureDoesNotMatch-equivalent)", rec.Code)
	}
}

func TestVerifier_SkewRejectedWithoutRPC(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	req := signedGET(t, h.clock().Add(-time.Hour)) // outside ±15m
	rec := do(h, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (RequestTimeTooSkewed-equivalent)", rec.Code)
	}
	if got := h.fake.calls.Load(); got != 0 {
		t.Errorf("RPCs = %d, want 0 (skew check precedes key lookup)", got)
	}
}

func TestVerifier_IAMOutageIs500(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	// Kill the backend before the first fetch: the failure must read as an
	// outage, not a denial, and must not be cached.
	closeSrv()

	rec := do(h, signedGET(t, h.clock()))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

// TestVerifier_FetchSurvivesCallerCancel pins the fix for a collapsed-waiter
// 500 burst: the singleflight leader's fetch used to run on the first
// caller's request context, so a client disconnecting mid-RPC canceled the
// fetch and failed every request collapsed onto it. A request must still
// verify successfully even if its own context is already canceled by the
// time the fetch runs.
func TestVerifier_FetchSurvivesCallerCancel(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	req := signedGET(t, h.clock())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req = req.WithContext(ctx)

	rec := do(h, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (fetch must survive caller cancellation), body=%s", rec.Code, rec.Body.String())
	}
	if got := h.fake.calls.Load(); got != 1 {
		t.Errorf("RPCs = %d, want 1", got)
	}
}

// TestVerifier_KnownAnswerFixture replays the worked example from the AWS
// SigV4 documentation (GET iam.amazonaws.com ListUsers, AKIDEXAMPLE,
// 20150830T123600Z) whose final signature AWS publishes, so the full
// canonicalization + derivation + comparison pipeline is checked against
// AWS's own reference vector rather than our own signer.
func TestVerifier_KnownAnswerFixture(t *testing.T) {
	h := &harness{now: time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC)}
	h.fake = &fakeKeys{cacheTTL: 5 * time.Minute, now: h.clock}
	srv := httptest.NewServer(stsv1.NewSigningKeyServiceServer(h.fake))
	defer srv.Close()
	v := New(Config{
		BaseURL:    srv.URL,
		HTTPClient: http.DefaultClient,
		Region:     "us-east-1",
		Service:    "iam",
		Now:        h.clock,
	})

	req := httptest.NewRequest(http.MethodGet, "https://iam.amazonaws.com/?Action=ListUsers&Version=2010-05-08", nil)
	req.Host = "iam.amazonaws.com"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("X-Amz-Date", "20150830T123600Z")
	req.Header.Set("Authorization", strings.Join([]string{
		"AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request",
		" SignedHeaders=content-type;host;x-amz-date",
		" Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7",
	}, ","))

	id, err := v.Verify(req)
	if err != nil {
		t.Fatalf("Verify(AWS doc fixture): %v", err)
	}
	if id.AccessKeyID != "AKIDEXAMPLE" {
		t.Errorf("AccessKeyID = %q", id.AccessKeyID)
	}
}
