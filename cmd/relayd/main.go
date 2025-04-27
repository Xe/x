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

	"github.com/avct/uasurfer"
	"github.com/google/uuid"
	"github.com/jackc/puddle/v2"
	_ "modernc.org/sqlite"
	"within.website/x/internal"
)

var (
	bind       = flag.String("bind", ":3004", "port to listen on")
	certDir    = flag.String("cert-dir", "/xe/pki", "where to read mounted certificates from")
	certFname  = flag.String("cert-fname", "tls.crt", "certificate filename")
	fpDatabase = flag.String("fp-database", "", "location of fingerprint database")
	keyFname   = flag.String("key-fname", "tls.key", "key filename")
	proxyTo    = flag.String("proxy-to", "http://localhost:5000", "where to reverse proxy to")
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

	uaParser, err := uaParserPool()
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

		// if tcpFP := GetTCPFingerprint(req); tcpFP != nil {
		// 	req.Header.Set("X-TCP-Fingerprint-JA4T", tcpFP.String())
		// }

		if uap, err := uaParser.Acquire(req.Context()); err == nil {
			defer uap.Release()

			uasurfer.ParseUserAgent(req.UserAgent(), uap.Value())

			vers := uap.Value().Browser.Version

			req.Header.Set("Xe-X-Relayd-UserAgent-Browser", uap.Value().Browser.Name.StringTrimPrefix())
			req.Header.Set("Xe-X-Relayd-UserAgent-Browser-Version", fmt.Sprintf("%d.%d.%d", vers.Major, vers.Minor, vers.Patch))
			req.Header.Set("Xe-X-Relayd-UserAgent-OS", uap.Value().OS.Name.StringTrimPrefix())
			req.Header.Set("Xe-X-Relayd-UserAgent-Platform", uap.Value().OS.Platform.StringTrimPrefix())
		}

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

func uaParserPool() (*puddle.Pool[*uasurfer.UserAgent], error) {
	cons := func(context.Context) (*uasurfer.UserAgent, error) {
		return &uasurfer.UserAgent{}, nil
	}
	des := func(ua *uasurfer.UserAgent) {
		ua.Reset()
	}

	pool, err := puddle.NewPool(&puddle.Config[*uasurfer.UserAgent]{
		Constructor: cons,
		Destructor:  des,
		MaxSize:     512,
	})

	return pool, err
}
