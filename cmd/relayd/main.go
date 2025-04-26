package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
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

	kpr, err := NewKeypairReloader(cert, key)
	if err != nil {
		log.Fatal(err)
	}

	h := httputil.NewSingleHostReverseProxy(u)
	oldDirector := h.Director

	if u.Scheme == "unix" {
		h = &httputil.ReverseProxy{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", strings.TrimPrefix(*proxyTo, "unix://"))
				},
			},
		}
	}

	h.Director = func(req *http.Request) {
		oldDirector(req)

		host, _, _ := net.SplitHostPort(req.RemoteAddr)
		if host != "" {
			req.Header.Set("X-Real-Ip", host)
		}

		fp := GetTLSFingerprint(req)
		if fp != nil {
			if fp.JA3N() != nil {
				req.Header.Set("X-TLS-Fingerprint-JA3N", fp.JA3N().String())
			}
			if fp.JA4() != nil {
				req.Header.Set("X-TLS-Fingerprint-JA4", fp.JA4().String())
			}
		}

		if tcpFP := GetTCPFingerprint(req); tcpFP != nil {
			req.Header.Set("X-TCP-Fingerprint-JA4T", tcpFP.String())
		}

		req.Header.Set("X-Forwarded-Host", req.URL.Host)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Scheme", "https")
		req.Header.Set("X-Request-Id", uuid.NewString())
		req.Header.Set("X-Scheme", "https")
		req.Header.Set("X-HTTP-Version", req.Proto)
	}

	srv := &http.Server{
		Addr:    *bind,
		Handler: h,
		TLSConfig: &tls.Config{
			GetCertificate: kpr.GetCertificate,
		},
	}

	applyTLSFingerprinter(srv)

	log.Fatal(srv.ListenAndServeTLS("", ""))
}

type keypairReloader struct {
	certMu   sync.RWMutex
	cert     *tls.Certificate
	certPath string
	keyPath  string
	modTime  time.Time
}

func NewKeypairReloader(certPath, keyPath string) (*keypairReloader, error) {
	result := &keypairReloader{
		certPath: certPath,
		keyPath:  keyPath,
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	result.cert = &cert

	st, err := os.Stat(certPath)
	if err != nil {
		return nil, err
	}
	result.modTime = st.ModTime()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)
		for range c {
			slog.Info("got SIGHUP")
			if err := result.maybeReload(); err != nil {
				slog.Error("can't load tls cert", "err", err)
			}
		}
	}()
	return result, nil
}

func (kpr *keypairReloader) maybeReload() error {
	slog.Info("loading new keypair", "cert", kpr.certPath, "key", kpr.keyPath)
	newCert, err := tls.LoadX509KeyPair(kpr.certPath, kpr.keyPath)
	if err != nil {
		return err
	}

	st, err := os.Stat(kpr.certPath)
	if err != nil {
		return err
	}

	kpr.certMu.Lock()
	defer kpr.certMu.Unlock()
	kpr.cert = &newCert
	kpr.modTime = st.ModTime()

	return nil
}

func (kpr *keypairReloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	kpr.certMu.RLock()
	defer kpr.certMu.RUnlock()

	st, err := os.Stat(kpr.certPath)
	if err != nil {
		return nil, fmt.Errorf("internal error: stat(%q): %q", kpr.certPath, err)
	}

	if st.ModTime().After(kpr.modTime) {
		kpr.certMu.RUnlock()
		if err := kpr.maybeReload(); err != nil {
			return nil, fmt.Errorf("can't reload cert: %w", err)
		}
		kpr.certMu.RLock()
	}

	return kpr.cert, nil
}
