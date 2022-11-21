package nodeinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const nodeInfo2point0 = `{"version":"2.0","software":{"name":"mastodon","version":"4.0.2"},"protocols":["activitypub"],"services":{"outbound":[],"inbound":[]},"usage":{"users":{"total":255,"activeMonth":163,"activeHalfyear":166},"localPosts":12107},"openRegistrations":true,"metadata":{}}`

func TestNodeInfo(t *testing.T) {
	mux := http.NewServeMux()
	s := httptest.NewServer(mux)
	mux.HandleFunc("/.well-known/nodeinfo", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(wellKnownLinks{Links: []wellKnownLink{
			{
				Rel:  schema2point0,
				Href: s.URL + "/nodeinfo/2.0",
			},
		}})
	})
	mux.HandleFunc("/nodeinfo/2.0", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, nodeInfo2point0)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := FetchWithClient(ctx, s.Client(), s.URL)
	if err != nil {
		t.Fatal(err)
	}
}
