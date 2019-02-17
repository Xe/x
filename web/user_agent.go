package web

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	http.DefaultTransport = &userAgentTransport{http.DefaultTransport}
}

func genUserAgent() string {
	return fmt.Sprintf("github.com-Xe-x (%s/%s/%s; %s/bot; +https://github.com/Xe/x/blob/master/web/x.md)", runtime.Version(), runtime.GOOS, runtime.GOARCH, filepath.Base(os.Args[0]))
}

type userAgentTransport struct {
	rt http.RoundTripper
}

func (uat userAgentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", genUserAgent())
	return uat.rt.RoundTrip(r)
}
