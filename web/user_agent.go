package web

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/celrenheit/sandflake"
)

var (
	startID  = sandflake.Next()
	hostname = "<unknown>"
	started  = time.Now()
)

func init() {
	http.DefaultTransport = &userAgentTransport{http.DefaultTransport}

	name, _ := os.Hostname()
	if name != "" {
		hostname = name
	}
}

// GenUserAgent creates a unique User-Agent string for outgoing HTTP requests.
func GenUserAgent() string {
	return fmt.Sprintf("github.com-Xe-x (%s/%s/%s; %s; +https://within.website/.x.botinfo) Alive (%s; sandflake) Hostname/%s Started (%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH, os.Args[0], startID.String(), hostname, started.Format(time.RFC3339))
}

type userAgentTransport struct {
	rt http.RoundTripper
}

func (uat userAgentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", GenUserAgent())
	return uat.rt.RoundTrip(r)
}
