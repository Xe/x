package twirpslog

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/twitchtv/twirp"
	"github.com/twitchtv/twirp/ctxsetters"
	"google.golang.org/protobuf/types/known/timestamppb"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	xslog "within.website/x/internal/slog"
	"within.website/x/web/middleware/sigv4"
	"within.website/x/web/middleware/sigv4/iamsts"
	"within.website/x/web/middleware/sigv4/sigv4client"
	"within.website/x/web/middleware/sigv4a"
	sigv4aiamsts "within.website/x/web/middleware/sigv4a/iamsts"
	"within.website/x/web/middleware/sigv4a/sigv4aclient"
)

// TestInterceptor checks the counting semantics: every call increments
// invocations, only failures increment errors, and only successes are billed
// to the caller.
func TestInterceptor(t *testing.T) {
	lg := slog.New(slog.DiscardHandler)

	// Each case uses its own method name so its counter series start at zero
	// on the shared default registry.
	newCtx := func(method string) context.Context {
		ctx := ctxsetters.WithPackageName(context.Background(), "test.pkg")
		ctx = ctxsetters.WithServiceName(ctx, "Svc")
		return ctxsetters.WithMethodName(ctx, method)
	}

	t.Run("success", func(t *testing.T) {
		m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
			return "resp", nil
		})
		resp, err := m(newCtx("Ok"), "req")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if resp != "resp" {
			t.Fatalf("resp = %v, want %q", resp, "resp")
		}

		if got := testutil.ToFloat64(invocations.WithLabelValues("test.pkg", "Svc", "Ok")); got != 1 {
			t.Errorf("invocations = %v, want 1", got)
		}
		if got := testutil.ToFloat64(errorHits.WithLabelValues("test.pkg", "Svc", "Ok")); got != 0 {
			t.Errorf("errors = %v, want 0", got)
		}
	})

	t.Run("error", func(t *testing.T) {
		wantErr := errors.New("boom")
		m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
			return nil, wantErr
		})
		if _, err := m(newCtx("Boom"), "req"); !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}

		if got := testutil.ToFloat64(invocations.WithLabelValues("test.pkg", "Svc", "Boom")); got != 1 {
			t.Errorf("invocations = %v, want 1", got)
		}
		if got := testutil.ToFloat64(errorHits.WithLabelValues("test.pkg", "Svc", "Boom")); got != 1 {
			t.Errorf("errors = %v, want 1", got)
		}
	})
}

// TestInterceptorContextAttrs proves that the interceptor attaches the call's
// package, service, and method to the context, so a service method that logs
// via slog.InfoContext surfaces them without threading a logger by hand.
func TestInterceptorContextAttrs(t *testing.T) {
	var buf bytes.Buffer
	lg := slog.New(xslog.NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	prev := slog.Default()
	slog.SetDefault(lg)
	t.Cleanup(func() { slog.SetDefault(prev) })

	ctx := ctxsetters.WithPackageName(context.Background(), "test.pkg")
	ctx = ctxsetters.WithServiceName(ctx, "Svc")
	ctx = ctxsetters.WithMethodName(ctx, "Attrs")

	m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
		// A handler deep inside the call logs through the default logger with
		// only the request context in hand.
		slog.InfoContext(ctx, "inside handler")
		return "resp", nil
	})
	if _, err := m(ctx, "req"); err != nil {
		t.Fatalf("err = %v", err)
	}

	var handlerLine map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("unmarshal %q: %v", line, err)
		}
		if rec["msg"] == "inside handler" {
			handlerLine = rec
		}
		// Each attribute key must appear exactly once per line: attaching the
		// attrs to both the logger and the context would emit duplicates that
		// json.Unmarshal silently collapses, so check the raw bytes.
		for _, key := range []string{`"package"`, `"service"`, `"method"`} {
			if n := bytes.Count(line, []byte(key)); n > 1 {
				t.Errorf("key %s appears %d times in %s", key, n, line)
			}
		}
	}
	if handlerLine == nil {
		t.Fatalf("no %q line in output:\n%s", "inside handler", buf.String())
	}

	for k, want := range map[string]string{
		"package": "test.pkg",
		"service": "Svc",
		"method":  "Attrs",
	} {
		if got := handlerLine[k]; got != want {
			t.Errorf("%s = %v, want %q", k, got, want)
		}
	}
}

// TestInterceptor_UserAttribution checks caller attribution for the two
// sources that are directly constructible from outside their packages:
// sigv4a.WithUser (iamd and other services authenticating locally with
// SigV4A) and the classic sigv4.WithUser (kept for other consumers). sigv4a
// must win when, hypothetically, both are present on the same context.
//
// The iamsts.Caller sources (both sigv4a and classic) are covered separately
// by TestInterceptor_IAMSTSCallerAttribution: their context key is unexported
// to their own package, so a caller identity can only be produced by driving
// a real signed request through that package's Middleware.
func TestInterceptor_UserAttribution(t *testing.T) {
	lg := slog.New(slog.DiscardHandler)

	newCtx := func(method string) context.Context {
		ctx := ctxsetters.WithPackageName(context.Background(), "test.pkg")
		ctx = ctxsetters.WithServiceName(ctx, "Svc")
		return ctxsetters.WithMethodName(ctx, method)
	}

	tests := []struct {
		name       string
		method     string
		buildCtx   func(ctx context.Context) context.Context
		wantUserID string
	}{
		{
			name:   "sigv4a user",
			method: "SigV4AUser",
			buildCtx: func(ctx context.Context) context.Context {
				return sigv4a.WithUser(ctx, &iamv1.User{Id: "sigv4a-user"})
			},
			wantUserID: "sigv4a-user",
		},
		{
			name:   "classic sigv4 user",
			method: "ClassicUser",
			buildCtx: func(ctx context.Context) context.Context {
				return sigv4.WithUser(ctx, &iamv1.User{Id: "classic-user"})
			},
			wantUserID: "classic-user",
		},
		{
			name:   "sigv4a user takes precedence over classic",
			method: "BothUsers",
			buildCtx: func(ctx context.Context) context.Context {
				ctx = sigv4.WithUser(ctx, &iamv1.User{Id: "classic-user"})
				return sigv4a.WithUser(ctx, &iamv1.User{Id: "sigv4a-user"})
			},
			wantUserID: "sigv4a-user",
		},
		{
			name:       "no caller resolved",
			method:     "NoCaller",
			buildCtx:   func(ctx context.Context) context.Context { return ctx },
			wantUserID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.buildCtx(newCtx(tt.method))
			m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
				return "resp", nil
			})
			if _, err := m(ctx, "req"); err != nil {
				t.Fatalf("err = %v", err)
			}

			label := fmt.Sprintf("test.pkg/Svc/%s", tt.method)
			if tt.wantUserID == "" {
				// No call attributes to any user id, so the label for an
				// empty user id was never incremented.
				if got := testutil.ToFloat64(usage.WithLabelValues(label, "")); got != 0 {
					t.Errorf("usage(%q, \"\") = %v, want 0", label, got)
				}
				return
			}
			if got := testutil.ToFloat64(usage.WithLabelValues(label, tt.wantUserID)); got != 1 {
				t.Errorf("usage(%q, %q) = %v, want 1", label, tt.wantUserID, got)
			}
		})
	}
}

// classicSigningKeyFake serves classic SigV4 derived signing keys for a
// single known credential — just enough to drive one real
// sigv4/iamsts.Verifier.Middleware round trip.
type classicSigningKeyFake struct {
	accessKeyID, secret, principalID, displayName string
}

func (f *classicSigningKeyFake) GetPublicKey(context.Context, *stsv1.GetPublicKeyRequest) (*stsv1.GetPublicKeyResponse, error) {
	return nil, twirp.NewError(twirp.Unimplemented, "not implemented in this fake")
}

func (f *classicSigningKeyFake) GetSigningKey(_ context.Context, req *stsv1.GetSigningKeyRequest) (*stsv1.GetSigningKeyResponse, error) {
	if req.GetAccessKeyId() != f.accessKeyID {
		return nil, twirp.NotFoundError("unknown access key id")
	}
	day, err := time.Parse("20060102", req.GetDate())
	if err != nil {
		return nil, twirp.InvalidArgumentError("date", "bad date")
	}
	nva := day.AddDate(0, 0, 1).Add(15 * time.Minute)
	return &stsv1.GetSigningKeyResponse{
		SigningKey: sigv4.DeriveSigningKey(f.secret, req.GetDate(), req.GetRegion(), req.GetService()),
		Identity: &stsv1.TokenIdentity{
			AccessKeyId: req.GetAccessKeyId(),
			PrincipalId: f.principalID,
			DisplayName: f.displayName,
		},
		NotValidAfter: timestamppb.New(nva),
		CacheUntil:    timestamppb.New(time.Now().Add(5 * time.Minute)),
	}, nil
}

// sigv4aSigningKeyFake serves SigV4A public keys for a single known
// credential — just enough to drive one real
// sigv4a/iamsts.Verifier.Middleware round trip.
type sigv4aSigningKeyFake struct {
	accessKeyID, secret, principalID, displayName string
}

func (f *sigv4aSigningKeyFake) GetSigningKey(context.Context, *stsv1.GetSigningKeyRequest) (*stsv1.GetSigningKeyResponse, error) {
	return nil, twirp.NewError(twirp.Unimplemented, "not implemented in this fake")
}

func (f *sigv4aSigningKeyFake) GetPublicKey(_ context.Context, req *stsv1.GetPublicKeyRequest) (*stsv1.GetPublicKeyResponse, error) {
	if req.GetAccessKeyId() != f.accessKeyID {
		return nil, twirp.NotFoundError("unknown access key id")
	}
	priv, err := sigv4a.DeriveKeyPair(req.GetAccessKeyId(), f.secret)
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
			PrincipalId: f.principalID,
			DisplayName: f.displayName,
		},
		CacheUntil: timestamppb.New(time.Now().Add(5 * time.Minute)),
	}, nil
}

// TestInterceptor_IAMSTSCallerAttribution drives a real signed request
// through each iamsts package's Middleware end to end, so the interceptor
// reads a genuine iamsts.Caller populated by that package rather than a
// context value assembled by hand (its key is unexported outside the
// package). It proves both the new sigv4a/iamsts branch this fix adds and
// the classic sigv4/iamsts branch it must not regress.
func TestInterceptor_IAMSTSCallerAttribution(t *testing.T) {
	const (
		accessKeyID = "AKIDEXAMPLE"
		secret      = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
		region      = "us-east-1"
		service     = "iam"
	)

	t.Run("sigv4a", func(t *testing.T) {
		fake := &sigv4aSigningKeyFake{accessKeyID: accessKeyID, secret: secret, principalID: "u-sigv4a", displayName: "sigv4a tester"}
		keySrv := httptest.NewServer(stsv1.NewSigningKeyServiceServer(fake))
		defer keySrv.Close()

		verifier := sigv4aiamsts.New(sigv4aiamsts.Config{
			BaseURL:     keySrv.URL,
			HTTPClient:  http.DefaultClient,
			Region:      region,
			Service:     service,
			MaxBodySize: 1 << 20,
		})

		lg := slog.New(slog.DiscardHandler)
		var callErr error
		app := httptest.NewServer(verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ctxsetters.WithPackageName(r.Context(), "test.pkg")
			ctx = ctxsetters.WithServiceName(ctx, "Svc")
			ctx = ctxsetters.WithMethodName(ctx, "SigV4AIAMSTS")
			m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
				return "resp", nil
			})
			_, callErr = m(ctx, "req")
			w.WriteHeader(http.StatusNoContent)
		})))
		defer app.Close()

		rt, err := sigv4aclient.NewSigV4ARoundTripper(&sigv4aclient.Config{
			Region: region, AccessKey: accessKeyID, SecretKey: secret, ServiceName: service,
		}, nil)
		if err != nil {
			t.Fatalf("round tripper: %v", err)
		}
		resp, err := (&http.Client{Transport: rt}).Get(app.URL + "/resource")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if callErr != nil {
			t.Fatalf("interceptor: %v", callErr)
		}

		if got := testutil.ToFloat64(usage.WithLabelValues("test.pkg/Svc/SigV4AIAMSTS", fake.principalID)); got != 1 {
			t.Errorf("usage(%q) = %v, want 1", fake.principalID, got)
		}
	})

	t.Run("classic", func(t *testing.T) {
		fake := &classicSigningKeyFake{accessKeyID: accessKeyID, secret: secret, principalID: "u-classic", displayName: "classic tester"}
		keySrv := httptest.NewServer(stsv1.NewSigningKeyServiceServer(fake))
		defer keySrv.Close()

		verifier := iamsts.New(iamsts.Config{
			BaseURL:     keySrv.URL,
			HTTPClient:  http.DefaultClient,
			Region:      region,
			Service:     service,
			MaxBodySize: 1 << 20,
		})

		lg := slog.New(slog.DiscardHandler)
		var callErr error
		app := httptest.NewServer(verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ctxsetters.WithPackageName(r.Context(), "test.pkg")
			ctx = ctxsetters.WithServiceName(ctx, "Svc")
			ctx = ctxsetters.WithMethodName(ctx, "ClassicIAMSTS")
			m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
				return "resp", nil
			})
			_, callErr = m(ctx, "req")
			w.WriteHeader(http.StatusNoContent)
		})))
		defer app.Close()

		rt, err := sigv4client.NewSigV4RoundTripper(&sigv4client.Config{
			Region: region, AccessKey: accessKeyID, SecretKey: secret, ServiceName: service,
		}, nil)
		if err != nil {
			t.Fatalf("round tripper: %v", err)
		}
		resp, err := (&http.Client{Transport: rt}).Get(app.URL + "/resource")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if callErr != nil {
			t.Fatalf("interceptor: %v", callErr)
		}

		if got := testutil.ToFloat64(usage.WithLabelValues("test.pkg/Svc/ClassicIAMSTS", fake.principalID)); got != 1 {
			t.Errorf("usage(%q) = %v, want 1", fake.principalID, got)
		}
	})
}
