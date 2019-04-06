package tun2

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestBackendAuthV1(t *testing.T) {
	st := MockStorage()

	s, err := NewServer(&ServerConfig{
		Storage: st,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	st.AddRoute(domain, user)
	st.AddToken(token, user, []string{"connect"})
	st.AddToken(noPermToken, user, nil)
	st.AddToken(otherUserToken, "cadey", []string{"connect"})

	cases := []struct {
		name    string
		auth    Auth
		wantErr bool
	}{
		{
			name: "basic everything should work",
			auth: Auth{
				Token:  token,
				Domain: domain,
			},
			wantErr: false,
		},
		{
			name: "invalid domain",
			auth: Auth{
				Token:  token,
				Domain: "aw.heck",
			},
			wantErr: true,
		},
		{
			name: "invalid token",
			auth: Auth{
				Token:  "asdfwtweg",
				Domain: domain,
			},
			wantErr: true,
		},
		{
			name: "invalid token scopes",
			auth: Auth{
				Token:  noPermToken,
				Domain: domain,
			},
			wantErr: true,
		},
		{
			name: "user token doesn't match domain owner",
			auth: Auth{
				Token:  otherUserToken,
				Domain: domain,
			},
			wantErr: true,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			data, err := json.Marshal(cs.auth)
			if err != nil {
				t.Fatal(err)
			}

			_, _, err = s.backendAuthv1(ctx, bytes.NewBuffer(data))

			if cs.wantErr && err == nil {
				t.Fatalf("auth did not err as expected")
			}

			if !cs.wantErr && err != nil {
				t.Fatalf("unexpected auth err: %v", err)
			}
		})
	}
}

func TestBackendRouting(t *testing.T) {
	t.Skip()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := MockStorage()

	st.AddRoute(domain, user)
	st.AddToken(token, user, []string{"connect"})

	s, err := NewServer(&ServerConfig{
		Storage: st,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	go s.Listen(l)

	cases := []struct {
		name           string
		wantStatusCode int
		handler        http.HandlerFunc
	}{
		{
			name:           "200 everything's okay",
			wantStatusCode: http.StatusOK,
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "HTTP 200, everything is okay :)", http.StatusOK)
			}),
		},
		{
			name:           "500 internal error",
			wantStatusCode: http.StatusInternalServerError,
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "HTTP 500, the world is on fire", http.StatusInternalServerError)
			}),
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			ts := httptest.NewServer(cs.handler)
			defer ts.Close()

			cc := &ClientConfig{
				ConnType:   "tcp",
				ServerAddr: l.Addr().String(),
				Token:      token,
				BackendURL: ts.URL,
				Domain:     domain,

				forceTCPClear: true,
			}

			c, err := NewClient(cc)
			if err != nil {
				t.Fatal(err)
			}

			go c.Connect(ctx) // TODO: fix the client library so this ends up actually getting cleaned up

			time.Sleep(125 * time.Millisecond)

			req, err := http.NewRequest("GET", "http://cetacean.club/", nil)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := s.RoundTrip(req)
			if err != nil {
				t.Fatalf("error in doing round trip: %v", err)
			}

			if cs.wantStatusCode != resp.StatusCode {
				resp.Write(os.Stdout)
				t.Fatalf("got status %d instead of %d", resp.StatusCode, cs.wantStatusCode)
			}
		})
	}
}

func setupTestServer() (*Server, *mockStorage, net.Listener, error) {
	st := MockStorage()

	st.AddRoute(domain, user)
	st.AddToken(token, user, []string{"connect"})

	s, err := NewServer(&ServerConfig{
		Storage: st,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	defer s.Close()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, nil, err
	}

	go s.Listen(l)

	return s, st, l, nil
}
