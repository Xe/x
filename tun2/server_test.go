package tun2

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"within.website/ln/opname"
)

// testing constants
const (
	user           = "shachi"
	token          = "orcaz r kewl"
	noPermToken    = "aw heck"
	otherUserToken = "even more heck"
	domain         = "cetacean.club"
)

func TestNewServerNullConfig(t *testing.T) {
	_, err := NewServer(nil)
	if err == nil {
		t.Fatalf("expected NewServer(nil) to fail, got non-failure")
	}
}

func TestGen502Page(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequest("GET", "http://cetacean.club", nil)
	if err != nil {
		t.Fatal(err)
	}

	substring := uuid.New()

	req = req.WithContext(ctx)
	req.Header.Add("X-Request-Id", substring)
	req.Host = "cetacean.club"

	resp := gen502Page(req)
	if resp == nil {
		t.Fatalf("expected response to be non-nil")
	}

	if resp.Body != nil {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(data), substring) {
			fmt.Println(string(data))
			t.Fatalf("502 page did not contain needed substring %q", substring)
		}
	}
}

func TestConnectionsCloseOnServerContextClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, _, l, err := setupTestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	defer l.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	for i := range make([]struct{}, 5) {
		ctx := opname.With(ctx, fmt.Sprint(i))
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
	}

	s.cancel()
	time.Sleep(125 * time.Millisecond)
	bes := s.GetAllBackends()
	if l := len(bes); l != 0 {
		t.Fatalf("expected len(bes) == 0, got: %d", l)
	}
}

func BenchmarkHTTP200(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, _, l, err := setupTestServer()
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	defer l.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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
		b.Fatal(err)
	}

	go c.Connect(ctx) // TODO: fix the client library so this ends up actually getting cleaned up

	for {
		r := s.GetBackendsForDomain(domain)
		if len(r) == 0 {
			time.Sleep(125 * time.Millisecond)
			continue
		}

		break
	}

	req, err := http.NewRequest("GET", "http://cetacean.club/", nil)
	if err != nil {
		b.Fatal(err)
	}

	_, err = s.RoundTrip(req)
	if err != nil {
		b.Fatalf("got error on initial request exchange: %v", err)
	}

	for n := 0; n < b.N; n++ {
		resp, err := s.RoundTrip(req)
		if err != nil {
			b.Fatalf("got error on %d: %v", n, err)
		}

		if resp.StatusCode != http.StatusOK {
			b.Fail()
			b.Logf("got %d instead of 200", resp.StatusCode)
		}
	}
}
