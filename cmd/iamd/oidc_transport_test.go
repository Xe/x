package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus/testutil"
	oidcv1 "within.website/x/gen/within/website/x/iam/oidc/v1"
	"within.website/x/gen/within/website/x/iam/oidc/v1/oidcv1connect"
)

func TestOIDCTransport(t *testing.T) {
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authorize":
			if r.FormValue("client_id") != "test" {
				t.Errorf("client id = %q", r.FormValue("client_id"))
			}
			http.SetCookie(w, &http.Cookie{Name: "login", Value: "opaque", HttpOnly: true, Secure: true})
			http.Redirect(w, r, "https://client.example/callback?code=one&state=state", http.StatusFound)
		case "/token":
			if r.FormValue("code") != "one" || r.Header.Get("Authorization") != "Basic encoded" {
				t.Errorf("lost form/credentials")
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":"invalid_grant"}`)
		case "/register":
			var doc map[string]any
			if err := json.NewDecoder(r.Body).Decode(&doc); err != nil || doc["client_name"] != "test" {
				t.Errorf("lost registration document: %v %v", doc, err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"client_id":"new"}`)
		case "/admin/resources":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"resources":[]}`)
		default:
			http.NotFound(w, r)
		}
	})
	admin := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "signed" {
				http.Error(w, "unauthenticated", 401)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	h, err := newOIDCTransport(core, admin)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, method, path, body, auth string
		status                         int
		contains                       string
	}{
		{"authorization redirect", "GET", "/authorize?client_id=test&state=state", "", "", 302, ""},
		{"form error", "POST", "/token", "grant_type=authorization_code&code=one", "Basic encoded", 400, `"error":"invalid_grant"`},
		{"registration", "POST", "/register", `{"client_name":"test"}`, "", 201, `"client_id":"new"`},
		{"admin needs signature", "GET", "/admin/resources", "", "", 401, ""},
		{"admin signature", "GET", "/admin/resources", "", "signed", 200, "resources"},
		{"duplicate code rejected", "POST", "/token", "code=one&code=two", "Basic encoded", 400, "invalid_request"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.method == "POST" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				if tt.path == "/register" {
					req.Header.Set("Content-Type", "application/json")
				}
			}
			req.Header.Set("Authorization", tt.auth)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tt.status {
				t.Fatalf("status %d want %d: %s", w.Code, tt.status, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tt.contains) {
				t.Fatalf("body %s missing %s", w.Body.String(), tt.contains)
			}
			if tt.status == 302 && (w.Header().Get("Location") == "" || len(w.Result().Cookies()) != 1) {
				t.Fatal("lost redirect/cookie")
			}
		})
	}
	ts := httptest.NewServer(h)
	defer ts.Close()
	client := oidcv1connect.NewProviderServiceClient(ts.Client(), ts.URL)
	req := connect.NewRequest(&oidcv1.ProtocolRequest{Code: "one", GrantType: "authorization_code"})
	req.Header().Set("Authorization", "Basic encoded")
	result, err := client.Token(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Msg.HttpStatus != 400 || result.Msg.Result.AsMap()["error"] != "invalid_grant" {
		t.Fatalf("native response = %v", result.Msg)
	}
}

func TestOIDCTransportLimitsAndBrowserMethods(t *testing.T) {
	calls := 0
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Method == http.MethodPost && r.FormValue("csrf") != "csrf" {
			t.Error("lost CSRF")
		}
		io.WriteString(w, "<!doctype html><title>Account</title>")
	})
	h, err := newOIDCTransport(core, func(h http.Handler) http.Handler { return h })
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			var body io.Reader
			if method == "POST" {
				body = strings.NewReader("csrf=csrf")
			}
			req := httptest.NewRequest(method, "/account", body)
			if method == "POST" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "<!doctype html>") {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
	before := calls
	req := httptest.NewRequest("POST", "/token", strings.NewReader("code="+strings.Repeat("x", 1<<20)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 400 || calls != before {
		t.Fatalf("oversized input: status=%d calls=%d", w.Code, calls)
	}
}

func TestOIDCTransportOriginAndCORS(t *testing.T) {
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "iamd.xeserv.us" {
			t.Errorf("host = %s", r.Host)
		}
		if r.RemoteAddr != "192.0.2.1:1234" {
			t.Errorf("remote address = %s", r.RemoteAddr)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	})
	h, err := newOIDCTransport(core, func(h http.Handler) http.Handler { return h })
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, method, path, body string
		status                   int
		cors                     bool
	}{
		{"REST origin", "GET", "/userinfo", "", 200, true},
		{"native peer", "POST", oidcv1connect.ProviderServiceRegisterProcedure, `{"document":{}}`, 200, true},
		{"SPA preflight", "OPTIONS", "/token", "", 204, true},
		{"no browser CORS", "OPTIONS", "/account", "", 405, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "https://iamd.xeserv.us"+tt.path, strings.NewReader(tt.body))
			req.RemoteAddr = "192.0.2.1:1234"
			req.Header.Set("Origin", "https://app.example")
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Connect-Protocol-Version", "1")
			}
			if tt.method == "OPTIONS" {
				req.Header.Set("Access-Control-Request-Method", "POST")
				req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tt.status {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tt.status, w.Body.String())
			}
			if (w.Header().Get("Access-Control-Allow-Origin") == "*") != tt.cors {
				t.Fatal("CORS policy missing or overly broad")
			}
			if w.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatal("must not allow credentialed CORS")
			}
		})
	}
}

func TestOIDCOperationMetrics(t *testing.T) {
	before := testutil.ToFloat64(oidcOperationRequests.WithLabelValues("/token", "400"))
	h, err := newOIDCTransport(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		io.WriteString(w, `{"error":"invalid_grant"}`)
	}), func(h http.Handler) http.Handler { return h })
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/token", strings.NewReader("code=bad"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), r)
	after := testutil.ToFloat64(oidcOperationRequests.WithLabelValues("/token", "400"))
	if after != before+1 {
		t.Fatalf("request count %v -> %v", before, after)
	}
}

func TestOIDCNativeBrowserReadAndMutation(t *testing.T) {
	var methods []string
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"csrf":"value"}`)
	})
	h, err := newOIDCTransport(core, func(h http.Handler) http.Handler { return h })
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	client := oidcv1connect.NewBrowserServiceClient(server.Client(), server.URL)
	for _, request := range []*oidcv1.ProtocolRequest{{}, {Action: "revoke_session", Csrf: "value"}} {
		if _, err := client.Account(context.Background(), connect.NewRequest(request)); err != nil {
			t.Fatal(err)
		}
	}
	if len(methods) != 2 || methods[0] != "GET" || methods[1] != "POST" {
		t.Fatalf("native methods=%v", methods)
	}
}

func TestOIDCExtensionsReachProtocolEngine(t *testing.T) {
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, key := range []string{"response_mode", "request", "request_uri", "client_secret"} {
			if r.FormValue(key) == "" {
				t.Errorf("lost %s", key)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		io.WriteString(w, `{"error":"invalid_request"}`)
	})
	h, err := newOIDCTransport(core, func(h http.Handler) http.Handler { return h })
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/authorize", strings.NewReader("response_mode=form_post&request=jwt&request_uri=https%3A%2F%2Fclient.example%2Fjwt&client_secret=unwanted"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("status=%d", w.Code)
	}
}
