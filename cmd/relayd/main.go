package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"within.website/x/internal"
)

var (
	bind      = flag.String("bind", ":3004", "port to listen on")
	certDir   = flag.String("cert-dir", "/xe/pki", "where to read mounted certificates from")
	certFname = flag.String("cert-fname", "tls.crt", "certificate filename")
	keyFname  = flag.String("key-fname", "tls.key", "key filename")
	proxyTo   = flag.String("proxy-to", "http://localhost:5000", "where to reverse proxy to")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting",
		"bind", *bind,
		"cert-dir", *certDir,
		"cert-fname", *certFname,
		"key-fname", *keyFname,
		"proxy-to", *proxyTo,
	)

	cert := filepath.Join(*certDir, *certFname)
	key := filepath.Join(*certDir, *keyFname)

	st, err := os.Stat(cert)

	if err != nil {
		slog.Error("can't stat cert file", "certFname", cert)
		os.Exit(1)
	}

	lastModified := st.ModTime()

	go func(lm time.Time) {
		t := time.NewTicker(time.Hour)
		defer t.Stop()

		for range t.C {
			st, err := os.Stat(cert)
			if err != nil {
				slog.Error("can't stat file", "fname", cert, "err", err)
				continue
			}

			if st.ModTime().After(lm) {
				slog.Info("new cert detected", "oldTime", lm.Format(time.RFC3339), "newTime", st.ModTime().Format(time.RFC3339))
				os.Exit(0)
			}
		}
	}(lastModified)

	u, err := url.Parse(*proxyTo)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(
		http.ListenAndServeTLS(
			*bind,
			cert,
			key,
			httputil.NewSingleHostReverseProxy(u),
		),
	)
}
