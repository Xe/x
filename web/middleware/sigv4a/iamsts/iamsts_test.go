package iamsts

import (
	"context"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4a"
)

const (
	testKey     = "AKIDEXAMPLE"
	testSecret  = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testKey2    = "AKIAI44QH8DHBEXAMPLE"
	testSecret2 = "je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY"
	testRegion  = "us-east-1"
	testSvc     = "execute-api"
)

// testSecretFor resolves the fake key service's known credentials.
func testSecretFor(accessKeyID string) (string, bool) {
	switch accessKeyID {
	case testKey:
		return testSecret, true
	case testKey2:
		return testSecret2, true
	default:
		return "", false
	}
}

// fakeKeys is a SigningKeyService stub served over a real Twirp endpoint. It
// derives real keys for the known test credentials, counts calls, and can be
// flipped to refuse. This fake only serves the public-key flow; GetSigningKey
// is unimplemented, mirroring the classic package's fake's stubbed
// GetPublicKey.
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

func (f *fakeKeys) GetSigningKey(context.Context, *stsv1.GetSigningKeyRequest) (*stsv1.GetSigningKeyResponse, error) {
	return nil, twirp.NewError(twirp.Unimplemented, "not implemented in this fake")
}

func (f *fakeKeys) GetPublicKey(_ context.Context, req *stsv1.GetPublicKeyRequest) (*stsv1.GetPublicKeyResponse, error) {
	f.calls.Add(1)
	f.mu.Lock()
	code := f.refuseCode
	f.mu.Unlock()
	if code != "" {
		return nil, twirp.NewError(code, "refused")
	}
	secret, ok := testSecretFor(req.GetAccessKeyId())
	if !ok {
		return nil, twirp.NotFoundError("unknown access key id")
	}
	priv, err := sigv4a.DeriveKeyPair(req.GetAccessKeyId(), secret)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &stsv1.GetPublicKeyResponse{
		PublicKey: der,
		Identity: &stsv1.TokenIdentity{
			AccessKeyId: req.GetAccessKeyId(),
			PrincipalId: "u1",
			DisplayName: "tester",
		},
		CacheUntil: timestamppb.New(f.now().Add(f.cacheTTL)),
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

// signedGETAs returns a bodyless GET for accessKeyID/secretAccessKey, signed
// at ts by the sigv4a signer.
func signedGETAs(t *testing.T, accessKeyID, secretAccessKey string, ts time.Time) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "https://svc.example.com/things?a=1", nil)
	s, err := sigv4a.NewSigner(accessKeyID, secretAccessKey, testRegion, testSvc)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	s.Now = func() time.Time { return ts }
	if err := s.Sign(req, nil); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return req
}

// signedGET returns a bodyless GET for the primary test credential.
func signedGET(t *testing.T, ts time.Time) *http.Request {
	t.Helper()
	return signedGETAs(t, testKey, testSecret, ts)
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

// TestVerifier_TTLSweep replaces the classic package's midnight-rollover
// test: SigV4A keys have no date/region/service scoping, so there is no
// day-boundary eviction to exercise. Instead this checks that inserting a
// second credential's entry sweeps a first credential's slot once its
// cache_until has passed.
func TestVerifier_TTLSweep(t *testing.T) {
	h, closeSrv := newHarness(t, time.Minute)
	defer closeSrv()

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("key1 request: status = %d", rec.Code)
	}
	if got := h.verifier.cacheLen(); got != 1 {
		t.Fatalf("cache slots after key1 = %d, want 1", got)
	}

	// Advance past key1's cache_until, then fetch key2: the insert must sweep
	// key1's now-expired slot.
	h.advance(2 * time.Minute)
	req2 := signedGETAs(t, testKey2, testSecret2, h.clock())
	if rec := do(h, req2); rec.Code != http.StatusNoContent {
		t.Fatalf("key2 request: status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := h.fake.calls.Load(); got != 2 {
		t.Errorf("RPCs = %d, want 2 (one per access key id)", got)
	}
	if got := h.verifier.cacheLen(); got != 1 {
		t.Errorf("cache slots = %d, want 1 (key1 evicted, key2 present)", got)
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
	// All n concurrent misses for one key must collapse into one RPC.
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
		t.Fatalf("status = %d, want 403 (signature-mismatch-equivalent)", rec.Code)
	}
}

func TestVerifier_SkewRejectedWithoutRPC(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	req := signedGET(t, h.clock().Add(-time.Hour)) // outside +-15m
	rec := do(h, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (clock-skew-equivalent)", rec.Code)
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
