package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
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
	"google.golang.org/protobuf/types/known/durationpb"
	_ "modernc.org/sqlite"
	"within.website/x/internal"
	"within.website/x/proto/relayd"
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
		"telemetry-enable", *telemetryEnable,
		"telemetry-bucket", *telemetryBucket,
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

	ts, err := NewTelemetrySink(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	if fpDatabase != nil {
		var err error
		db, err = sql.Open("sqlite", *fpDatabase)
		if err != nil {
			log.Fatal(err)
		}
		if err := db.Ping(); err != nil {
			log.Fatal(err)
		}

		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS fingerprints
			( "application" TEXT
			, "user_agent_string" TEXT
			, "notes" TEXT
			, "ja4_fingerprint" TEXT
			, ip_address TEXT
			, headers TEXT
			)`); err != nil {
			log.Fatal(err)
		}

		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS request_logs
			( "timestamp" TEXT
			, "method" TEXT
			, "path" TEXT
			, "query" TEXT
			, "ip_address" TEXT
			, "ja3n" TEXT
			, "ja4" TEXT
			, "headers" TEXT
			, "request_time" TEXT
			)`); err != nil {
			log.Fatal(err)
		}
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t0 := time.Now()

		var foundJa3n, foundJa4 string

		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host != "" {
			r.Header.Set("X-Real-Ip", host)
		}

		fp := GetTLSFingerprint(r)
		if fp != nil {
			if fp.JA3N() != nil {
				foundJa3n = fp.JA3N().String()
			}
			if fp.JA4() != nil {
				foundJa4 = fp.JA4().String()
			}
		}

		reqID := uuid.Must(uuid.NewV7()).String()
		rl := relayd.RequestLogFromRequest(r, host, reqID, foundJa3n, foundJa4)

		r.Header.Set("X-Forwarded-Host", r.URL.Host)
		r.Header.Set("X-Forwarded-Proto", "https")
		r.Header.Set("X-Forwarded-Scheme", "https")
		r.Header.Set("X-Request-Id", reqID)
		r.Header.Set("X-Scheme", "https")
		r.Header.Set("X-HTTP-Protocol", r.Proto)
		r.Header.Set("X-TLS-Fingerprint-JA3N", foundJa3n)
		r.Header.Set("X-TLS-Fingerprint-JA4", foundJa4)

		headers, _ := json.Marshal(r.Header)

		if ja4 := foundJa4; db != nil && ja4 != "" {
			var application, userAgent, notes sql.NullString
			if err := db.QueryRowContext(r.Context(), "SELECT application, user_agent_string, notes FROM fingerprints WHERE ja4_fingerprint = ?", ja4).Scan(&application, &userAgent, &notes); err == nil {
				slog.Debug("found a hit", "application", application, "userAgent", userAgent, "notes", notes)
			} else if errors.Is(err, sql.ErrNoRows) {
				userAgent := r.UserAgent()
				notes := fmt.Sprintf("Observed via relayd on host %s at %s", r.Host, time.Now().Format(time.RFC3339))
				if _, err := db.ExecContext(r.Context(), "INSERT INTO fingerprints(user_agent_string, notes, ja4_fingerprint, ip_address, headers) VALUES (?, ?, ?, ?, ?)", userAgent, notes, ja4, host, string(headers)); err != nil {
					slog.Error("can't insert fingerprint into database", "err", err)
				}

				r.Header.Set("Xe-X-Relayd-New-Client", "true")
			} else {
				slog.Debug("can't read from database", "err", err)
			}

			rp.ServeHTTP(w, r)

			done := time.Since(t0)
			rl.ResponseTime = durationpb.New(done)

			if ts != nil {
				ts.Add(rl)
			}
		}
	})

	if u.Scheme == "unix" {
		rp = &httputil.ReverseProxy{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", strings.TrimPrefix(*proxyTo, "unix://"))
				},
			},
		}
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
