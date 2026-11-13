package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"connectrpc.com/connect"
	"connectrpc.com/vanguard"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	oidcv1 "within.website/x/gen/within/website/x/iam/oidc/v1"
	"within.website/x/gen/within/website/x/iam/oidc/v1/oidcv1connect"
)

// oidcWire is local context, not client-provided metadata. It preserves the
// original method and lets the protocol operation select the REST status.
type oidcWire struct {
	method, remoteAddr, host string
	status                   int
}
type oidcWireKey struct{}

var oidcOperationRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "iamd_oidc_requests_total", Help: "OIDC protocol outcomes by operation and HTTP status.",
}, []string{"operation", "status"})
var oidcOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "iamd_oidc_request_duration_seconds", Help: "OIDC protocol operation duration.",
}, []string{"operation"})

type oidcAdapter struct{ core http.Handler }

// newOIDCTransport keeps all public protocol and browser operations behind
// generated Connect handlers. Only wire encoding and status adaptation happen
// outside Connect. Administration authenticates the original bytes first.
func newOIDCTransport(core http.Handler, admin func(http.Handler) http.Handler) (http.Handler, error) {
	adapter := &oidcAdapter{core: core}
	providerPath, provider := oidcv1connect.NewProviderServiceHandler(adapter, connect.WithReadMaxBytes(1<<20))
	browserPath, browser := oidcv1connect.NewBrowserServiceHandler(adapter, connect.WithReadMaxBytes(1<<20))
	adminPath, administration := oidcv1connect.NewAdministrationServiceHandler(adapter, connect.WithReadMaxBytes(1<<20))
	transcoder, err := vanguard.NewTranscoder([]*vanguard.Service{
		vanguard.NewService(providerPath, provider, vanguard.WithMaxMessageBufferBytes(1<<20)),
		vanguard.NewService(browserPath, browser, vanguard.WithMaxMessageBufferBytes(1<<20)),
		vanguard.NewService(adminPath, administration, vanguard.WithMaxMessageBufferBytes(1<<20)),
	})
	if err != nil {
		return nil, fmt.Errorf("OIDC transcoder: %w", err)
	}
	wire := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/within.website.x.iam.oidc.v1.") {
			state := &oidcWire{remoteAddr: r.RemoteAddr, host: r.Host}
			r = r.WithContext(context.WithValue(r.Context(), oidcWireKey{}, state))
			transcoder.ServeHTTP(w, r)
			return
		}
		state := &oidcWire{method: r.Method, remoteAddr: r.RemoteAddr, host: r.Host}
		r = r.WithContext(context.WithValue(r.Context(), oidcWireKey{}, state))
		if err := normalizeOIDCRequest(r); err != nil {
			writeOIDCWireError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		// Vanguard's HTTP mapping has a fixed success status. Buffer its unary
		// response so OAuth's status and error contract are applied atomically.
		rec := httptest.NewRecorder()
		transcoder.ServeHTTP(rec, r)
		if state.status == 0 && rec.Code >= 400 {
			code := "invalid_request"
			if rec.Code >= 500 {
				code = "server_error"
			}
			writeOIDCWireError(w, rec.Code, code)
			return
		}
		for k, values := range rec.Header() {
			if strings.EqualFold(k, "Content-Length") {
				continue
			}
			w.Header()[k] = append([]string(nil), values...)
		}
		status := state.status
		if status == 0 {
			status = rec.Code
		}
		w.WriteHeader(status)
		if status != http.StatusNoContent {
			_, _ = w.Write(rec.Body.Bytes())
		}
	})
	protected := admin(wire)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if oidcPublicCORS(r.URL.Path) && r.Header.Get("Origin") != "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, X-User-Agent")
			w.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/admin/") || strings.HasPrefix(r.URL.Path, adminPath) {
			protected.ServeHTTP(w, r)
			return
		}
		wire.ServeHTTP(w, r)
	}), nil
}

func writeOIDCWireError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

func normalizeOIDCRequest(r *http.Request) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	// Protocol requests have scalar parameters: duplicate security-sensitive
	// inputs and repeated resource requests are rejected rather than collapsed.
	fields := (&oidcv1.ProtocolRequest{}).ProtoReflect().Descriptor().Fields()
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return err
	}
	for key, values := range query {
		if len(values) != 1 {
			return fmt.Errorf("duplicate parameter")
		}
		if fields.ByName(protoreflect.Name(key)) == nil {
			delete(query, key)
		}
	}
	r.URL.RawQuery = query.Encode()
	if r.Method != http.MethodPost {
		return nil
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return err
	}
	if media != "application/x-www-form-urlencoded" {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return err
	}
	document := map[string]string{}
	for key, entries := range values {
		if len(entries) != 1 || query.Has(key) {
			return fmt.Errorf("duplicate parameter")
		}
		if fields.ByName(protoreflect.Name(key)) != nil {
			document[key] = entries[0]
		}
	}
	data, err := json.Marshal(document)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	r.ContentLength = int64(len(data))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Del("Content-Length")
	return nil
}

func (a *oidcAdapter) protocol(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest], method, path string) (*connect.Response[oidcv1.ProtocolResponse], error) {
	values := url.Values{}
	req.Msg.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		values.Set(string(field.Name()), value.String())
		return true
	})
	if state, ok := ctx.Value(oidcWireKey{}).(*oidcWire); ok && state.method != "" {
		method = state.method
	}
	var body io.Reader
	if method == http.MethodGet {
		path += "?" + values.Encode()
	} else {
		body = strings.NewReader(values.Encode())
	}
	r, err := http.NewRequestWithContext(ctx, method, "https://iamd.invalid"+path, body)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	r.Header = req.Header().Clone()
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if state, ok := ctx.Value(oidcWireKey{}).(*oidcWire); !ok || state.method == "" {
		r.Header.Set("Accept", "application/json")
	}
	return a.execute(ctx, r)
}
func (a *oidcAdapter) document(ctx context.Context, req *connect.Request[oidcv1.DocumentRequest], path string) (*connect.Response[oidcv1.ProtocolResponse], error) {
	if req.Msg.Document == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("document is required"))
	}
	data, err := protojson.Marshal(req.Msg.Document)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://iamd.invalid"+path, bytes.NewReader(data))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	r.Header = req.Header().Clone()
	r.Header.Set("Content-Type", "application/json")
	return a.execute(ctx, r)
}
func (a *oidcAdapter) execute(ctx context.Context, r *http.Request) (*connect.Response[oidcv1.ProtocolResponse], error) {
	if state, ok := ctx.Value(oidcWireKey{}).(*oidcWire); ok {
		r.RemoteAddr = state.remoteAddr
		r.Host = state.host
	}
	rec := httptest.NewRecorder()
	started := time.Now()
	a.core.ServeHTTP(rec, r)
	oidcOperationRequests.WithLabelValues(r.URL.Path, strconv.Itoa(rec.Code)).Inc()
	oidcOperationDuration.WithLabelValues(r.URL.Path).Observe(time.Since(started).Seconds())
	result := &oidcv1.ProtocolResponse{HttpStatus: int32(rec.Code), Body: &httpbody.HttpBody{ContentType: rec.Header().Get("Content-Type"), Data: rec.Body.Bytes()}}
	if strings.HasPrefix(result.Body.ContentType, "application/json") && len(result.Body.Data) > 0 {
		result.Result = &structpb.Struct{}
		if err := protojson.Unmarshal(result.Body.Data, result.Result); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid protocol JSON response"))
		}
	}
	if state, ok := ctx.Value(oidcWireKey{}).(*oidcWire); ok {
		state.status = rec.Code
	}
	response := connect.NewResponse(result)
	for k, values := range rec.Header() {
		switch strings.ToLower(k) {
		case "content-type", "content-length", "transfer-encoding", "connection", "trailer":
			continue
		}
		response.Header()[k] = append([]string(nil), values...)
	}
	return response, nil
}

func (a *oidcAdapter) Discovery(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/.well-known/openid-configuration")
}

func (a *oidcAdapter) AuthorizationServerMetadata(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/.well-known/oauth-authorization-server")
}

func (a *oidcAdapter) Keys(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/jwks.json")
}

func (a *oidcAdapter) Authorize(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/authorize")
}

func (a *oidcAdapter) Token(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "POST", "/token")
}

func (a *oidcAdapter) UserInfo(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/userinfo")
}

func (a *oidcAdapter) Revoke(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "POST", "/revoke")
}

func (a *oidcAdapter) UpstreamCallback(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/upstream/callback")
}

func (a *oidcAdapter) BackchannelLogout(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "POST", "/upstream/backchannel-logout")
}

func (a *oidcAdapter) Consent(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	method := "GET"
	if req.Msg.Decision != "" {
		method = "POST"
	}
	return a.protocol(ctx, req, method, "/consent")
}

func (a *oidcAdapter) Account(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	method := "GET"
	if req.Msg.Action != "" || req.Msg.Csrf != "" {
		method = "POST"
	}
	return a.protocol(ctx, req, method, "/account")
}

func (a *oidcAdapter) Logout(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	method := "GET"
	if req.Msg.Csrf != "" {
		method = "POST"
	}
	return a.protocol(ctx, req, method, "/logout")
}

func (a *oidcAdapter) ListResources(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/admin/resources")
}

func (a *oidcAdapter) ListClients(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/admin/clients")
}

func (a *oidcAdapter) RotateKeys(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "POST", "/admin/keys/rotate")
}

func (a *oidcAdapter) Register(ctx context.Context, req *connect.Request[oidcv1.DocumentRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.document(ctx, req, "/register")
}

func (a *oidcAdapter) PutResource(ctx context.Context, req *connect.Request[oidcv1.DocumentRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.document(ctx, req, "/admin/resources")
}

func (a *oidcAdapter) DisableClient(ctx context.Context, req *connect.Request[oidcv1.DocumentRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.document(ctx, req, "/admin/clients")
}

func (a *oidcAdapter) LinkIdentity(ctx context.Context, req *connect.Request[oidcv1.DocumentRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.document(ctx, req, "/admin/identities")
}

func (a *oidcAdapter) Login(ctx context.Context, req *connect.Request[oidcv1.ProtocolRequest]) (*connect.Response[oidcv1.ProtocolResponse], error) {
	return a.protocol(ctx, req, "GET", "/login")
}

// Public API requests use bearer/client credentials rather than browser cookies.
// Credentialed CORS is deliberately absent; session/admin endpoints are excluded.
func oidcPublicCORS(path string) bool {
	switch path {
	case "/.well-known/openid-configuration", "/.well-known/oauth-authorization-server", "/jwks.json", "/token", "/userinfo", "/register", "/revoke",
		oidcv1connect.ProviderServiceDiscoveryProcedure, oidcv1connect.ProviderServiceAuthorizationServerMetadataProcedure, oidcv1connect.ProviderServiceKeysProcedure,
		oidcv1connect.ProviderServiceTokenProcedure, oidcv1connect.ProviderServiceUserInfoProcedure, oidcv1connect.ProviderServiceRegisterProcedure, oidcv1connect.ProviderServiceRevokeProcedure:
		return true
	default:
		return false
	}
}
