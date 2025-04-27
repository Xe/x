package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
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
	"golang.org/x/crypto/acme/autocert"
	_ "modernc.org/sqlite"
	"within.website/x/internal"
	"within.website/x/xess"
)

var (
	useAutocert         = flag.Bool("use-autocert", false, "if true, provision certs with autocert")
	autocertCacheDir    = flag.String("autocert-cache-dir", "", "location to store cached certificates and ACME account details")
	autocertDomainNames = flag.String("autocert-domain-names", "", "comma-separated list of TLS hostnames to allow")
	autocertEmail       = flag.String("autocert-email", "", "ACME account contact email")
	bind                = flag.String("bind", ":3004", "port to listen on")
	certDir             = flag.String("cert-dir", "/xe/pki", "where to read mounted certificates from")
	certFname           = flag.String("cert-fname", "tls.crt", "certificate filename")
	fpDatabase          = flag.String("fp-database", "", "location of fingerprint database")
	keyFname            = flag.String("key-fname", "tls.key", "key filename")
	httpBind            = flag.String("http-bind", "", "if set, plain HTTP port to listen on to forward requests to https")
	proxyTo             = flag.String("proxy-to", "http://localhost:5000", "where to reverse proxy to")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting",
		"bind", *bind,
		"cert-dir", *certDir,
		"cert-fname", *certFname,
		"fp-database", *fpDatabase,
		"key-fname", *keyFname,
		"proxy-to", *proxyTo,
		"use-autocert", *useAutocert,
		"autocert-cache-dir", *autocertCacheDir,
		"autocert-domain-names", *autocertDomainNames,
		"autocert-email", *autocertEmail,
	)

	var tc *tls.Config

	switch *useAutocert {
	case true:
		slog.Info("using autocert")
		var fail bool
		if *autocertCacheDir == "" {
			fmt.Fprintln(os.Stderr, "cannot use --autocert without --autocert-cache-dir")
			fail = true
		}
		if *autocertDomainNames == "" {
			fmt.Fprintln(os.Stderr, "cannot use --autocert without --autocert-domain-names")
			fail = true
		}
		if *autocertEmail == "" {
			fmt.Fprintln(os.Stderr, "cannot use --autocert without --autocert-email")
			fail = true
		}
		if *httpBind != ":80" {
			fmt.Fprintln(os.Stderr, "cannot use --autocert without --http-bind=:80")
			fail = true
		}

		if fail {
			log.Fatal("autocert configuration errors")
		}

		m := &autocert.Manager{
			Cache:      autocert.DirCache(*autocertCacheDir),
			Prompt:     autocert.AcceptTOS,
			Email:      *autocertEmail,
			HostPolicy: autocert.HostWhitelist(strings.Split(*autocertDomainNames, ",")...),
		}

		go func() {
			slog.Info("listening for plain HTTP", "http-bind", *httpBind)
			log.Fatal(http.ListenAndServe(*httpBind, m.HTTPHandler(http.HandlerFunc(xess.NotFound))))
		}()

		tc = m.TLSConfig()
	case false:
		cert := filepath.Join(*certDir, *certFname)
		key := filepath.Join(*certDir, *keyFname)

		kpr, err := NewKeypairReloader(cert, key)
		if err != nil {
			log.Fatal(err)
		}
		tc = &tls.Config{GetCertificate: kpr.GetCertificate}
	}

	u, err := url.Parse(*proxyTo)
	if err != nil {
		log.Fatal(err)
	}

	var db *sql.DB

	if fpDatabase != nil {
		var err error
		db, err = sql.Open("sqlite", *fpDatabase)
		if err != nil {
			log.Fatal(err)
		}
		if err := db.Ping(); err != nil {
			log.Fatal(err)
		}
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

		// if tcpFP := GetTCPFingerprint(req); tcpFP != nil {
		// 	req.Header.Set("X-TCP-Fingerprint-JA4T", tcpFP.String())
		// }

		if ja4 := req.Header.Get("X-TLS-Fingerprint-JA4"); db != nil && ja4 != "" {
			var application, userAgent, notes sql.NullString
			if err := db.QueryRowContext(req.Context(), "SELECT application, user_agent_string, notes FROM fingerprints WHERE ja4_fingerprint = ?", ja4).Scan(&application, &userAgent, &notes); err == nil {
				slog.Debug("found a hit", "application", application, "userAgent", userAgent, "notes", notes)
				if application.Valid {
					req.Header.Set("Xe-X-Relayd-Ja4-Application", application.String)
				}

				if userAgent.Valid {
					req.Header.Set("Xe-X-Relayd-Ja4-UserAgent", userAgent.String)
				}

				if notes.Valid {
					req.Header.Set("Xe-X-Relayd-Ja4-Notes", notes.String)
				}
			} else if errors.Is(err, sql.ErrNoRows) {
				userAgent := req.UserAgent()
				notes := fmt.Sprintf("Observed via relayd on host %s at %s", req.Host, time.Now().Format(time.RFC3339))
				if _, err := db.ExecContext(req.Context(), "INSERT INTO fingerprints(user_agent_string, notes, ja4_fingerprint) VALUES (?, ?, ?)", userAgent, notes, ja4); err != nil {
					slog.Error("can't insert fingerprint into database", "err", err)
				}

				req.Header.Set("Xe-X-Relayd-New-Client", "true")
			} else {
				slog.Debug("can't read from database", "err", err)
			}
		}

		req.Header.Set("X-Forwarded-Host", req.URL.Host)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Scheme", "https")
		req.Header.Set("X-Request-Id", uuid.NewString())
		req.Header.Set("X-Scheme", "https")
		req.Header.Set("X-HTTP-Version", req.Proto)
	}

	srv := &http.Server{
		Addr:      *bind,
		Handler:   h,
		TLSConfig: tc,
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
