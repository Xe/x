package useragent

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
)

var (
	hostname = "<unknown>"
)

func init() {
	name, _ := os.Hostname()
	if name != "" {
		hostname = name
	}
}

// GenUserAgent creates a unique User-Agent string for outgoing HTTP requests.
func GenUserAgent(prefix, infoURL string) string {
	return fmt.Sprintf(
		"%s (%s/%s/%s; %s; +%s) Hostname/%s",
		prefix, runtime.Version(), runtime.GOOS, runtime.GOARCH, infoURL,
		os.Args[0], hostname,
	)
}

// Transport wraps a http transport with user agent information.
func Transport(prefix, infoURL string, rt http.RoundTripper) http.RoundTripper {
	return userAgentTransport{prefix: prefix, rt: rt}
}

type userAgentTransport struct {
	prefix, infoURL string
	rt              http.RoundTripper
}

func (uat userAgentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", GenUserAgent(uat.prefix, uat.infoURL))
	return uat.rt.RoundTrip(r)
}
