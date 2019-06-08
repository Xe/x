package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"within.website/x/internal"
)

func fwdhttps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST", "PUT", "PATCH":
		http.Error(w, "HTTPS access required", 400)
		return
	default:
		http.RedirectHandler(fmt.Sprintf("https://%s%s", r.Host, r.RequestURI), http.StatusPermanentRedirect).ServeHTTP(w, r)
	}
}

var (
	insecurePort = flag.String("insecure-bind", ":80", "host/port to bind on for insecure (HTTP) traffic")
	securePort   = flag.String("secure-bind", ":443", "host/port to bind on for secure (HTTPS) traffic")
	sitePort     = flag.String("site-port", "3000", "port to http forward")
	siteDomain   = flag.String("site-domain", "git.xeserv.us", "site port")
)

func main() {
	internal.HandleStartup()

	go http.ListenAndServe(*insecurePort, http.HandlerFunc(fwdhttps))

	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*siteDomain),
		Cache:      autocert.DirCache("./.relayd"),
	}

	u, err := url.Parse("http://127.0.0.1:" + *sitePort)
	if err != nil {
		panic(err)
	}

	go func() {
		err := http.ListenAndServe(*insecurePort, m.HTTPHandler(http.HandlerFunc(http.NotFound)))
		if err != nil {
			log.Fatal(err)
		}
	}()

	rp := httputil.NewSingleHostReverseProxy(u)

	s := &http.Server{
		IdleTimeout: 5 * time.Minute,
		Addr:        *securePort,
		TLSConfig:   &tls.Config{GetCertificate: m.GetCertificate},
		Handler:     rp,
	}
	s.ListenAndServeTLS("", "")
}

func checkCert(ctx context.Context, host string) error {
	if host == *siteDomain {
		return nil
	}

	return errors.New("not allowed")
}
