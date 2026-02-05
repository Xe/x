package entrypoint

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/felixge/httpsnoop"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/lum8rjack/go-ja4h"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/natefinch/lumberjack.v2"
	"within.website/x/autocert/s3cache"
	"within.website/x/cmd/sakurajima/internal/config"
	"within.website/x/cmd/sakurajima/internal/logging"
	"within.website/x/cmd/sakurajima/internal/logging/expressions"
	"within.website/x/fingerprint"
)

var (
	ErrTargetInvalid     = errors.New("[unexpected] target invalid")
	ErrNoHandler         = errors.New("[unexpected] no handler for domain")
	ErrInvalidTLSKeypair = errors.New("[unexpected] invalid TLS keypair")
	ErrNoCert            = errors.New("this server does not have a certificate for that domain")

	requestsPerDomain = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "techaro",
		Subsystem: "osiris",
		Name:      "request_count",
	}, []string{"domain", "method", "response_code"})

	responseTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "techaro",
		Subsystem: "osiris",
		Name:      "response_time",
	}, []string{"domain"})

	unresolvedRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "techaro",
		Subsystem: "osiris",
		Name:      "unresolved_requests",
	})
)

type Router struct {
	lock                 sync.RWMutex
	routes               map[string]http.Handler
	tlsCerts             map[string]*tls.Certificate
	opts                 Options
	accessLog            atomic.Value // stores *lumberjack.Logger
	baseSlog             *slog.Logger
	log                  atomic.Value // stores *slog.Logger
	autoMgr              *autocert.Manager
	autocertRedirectCode int
}

func (rtr *Router) setConfig(c config.Toplevel) error {
	var errs []error
	newMap := map[string]http.Handler{}
	newCerts := map[string]*tls.Certificate{}

	// Build host policy list for autocert based on domains with tls.autocert=true
	var autocertHosts []string

	for _, d := range c.Domains {
		var domainErrs []error

		u, err := url.Parse(d.Target)
		if err != nil {
			domainErrs = append(domainErrs, fmt.Errorf("%w %q: %v", ErrTargetInvalid, d.Target, err))
		}

		var h http.Handler

		if u != nil {
			switch u.Scheme {
			case "http", "https":
				rp := httputil.NewSingleHostReverseProxy(u)

				if d.InsecureSkipVerify {
					rp.Transport = &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					}
				}

				h = rp
			case "h2c":
				h = newH2CReverseProxy(u)
			case "unix":
				h = &httputil.ReverseProxy{
					Director: func(r *http.Request) {
						r.URL.Scheme = "http"
						r.URL.Host = d.Name
						r.Host = d.Name
					},
					Transport: &http.Transport{
						DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
							return net.Dial("unix", strings.TrimPrefix(d.Target, "unix://"))
						},
					},
				}
			}
		}

		if h == nil {
			domainErrs = append(domainErrs, ErrNoHandler)
		}

		newMap[d.Name] = h

		if d.TLS.Autocert {
			autocertHosts = append(autocertHosts, d.Name)
		} else {
			cert, err := tls.LoadX509KeyPair(d.TLS.Cert, d.TLS.Key)
			if err != nil {
				domainErrs = append(domainErrs, fmt.Errorf("%w: %w", ErrInvalidTLSKeypair, err))
			}
			newCerts[d.Name] = &cert
		}

		if len(domainErrs) != 0 {
			errs = append(errs, fmt.Errorf("invalid domain %s: %w", d.Name, errors.Join(domainErrs...)))
		}
	}

	if len(errs) != 0 {
		return fmt.Errorf("can't compile config to routing map: %w", errors.Join(errs...))
	}

	// Initialize autocert manager if needed
	var autoMgr *autocert.Manager
	if len(autocertHosts) > 0 {
		cache, err := s3cache.New(context.Background(), s3cache.Options{Bucket: c.Autocert.S3Bucket, Prefix: c.Autocert.S3Prefix})
		if err != nil {
			return fmt.Errorf("failed to init s3 autocert cache: %w", err)
		}
		autoMgr = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(autocertHosts...),
			Cache:      cache,
			Email:      c.Autocert.Email,
		}

		if c.Autocert.DirectoryURL != "" {
			autoMgr.Client = &acme.Client{DirectoryURL: c.Autocert.DirectoryURL}
		}
	}

	if oldLog := rtr.accessLog.Load(); oldLog != nil {
		if oldLogger, ok := oldLog.(*lumberjack.Logger); ok && oldLogger != nil {
			oldLogger.Rotate()
			oldLogger.Close()
		}
	}

	lum := &lumberjack.Logger{
		Filename:   c.Logging.AccessLog,
		MaxSize:    c.Logging.MaxSizeMB,
		MaxAge:     c.Logging.MaxAgeDays,
		MaxBackups: c.Logging.MaxBackups,
		Compress:   c.Logging.Compress,
	}

	var filters []logging.LogFilter

	h := rtr.baseSlog.Handler()

	for _, f := range c.Logging.Filters {
		filter, err := expressions.NewFilter(rtr.baseSlog, f.Name, f.Expression)
		if err != nil {
			return fmt.Errorf("can't compile filter expression: %w", err)
		}

		filters = append(filters, filter.Filter)
	}

	if len(filters) != 0 {
		h = logging.NewFilteringHandler(h, filters...)
	}

	log := slog.New(h)

	rtr.lock.Lock()
	rtr.routes = newMap
	rtr.tlsCerts = newCerts
	rtr.accessLog.Store(lum)
	rtr.log.Store(log)
	rtr.autoMgr = autoMgr
	if c.Autocert != nil && c.Autocert.HTTPRedirectCode != 0 {
		rtr.autocertRedirectCode = c.Autocert.HTTPRedirectCode
	} else {
		rtr.autocertRedirectCode = http.StatusMovedPermanently
	}
	rtr.lock.Unlock()

	return nil
}

func (rtr *Router) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	rtr.lock.RLock()
	cert, ok := rtr.tlsCerts[hello.ServerName]
	auto := rtr.autoMgr
	rtr.lock.RUnlock()

	if ok {
		return cert, nil
	}

	if auto != nil {
		// Delegate to autocert manager for configured hosts
		return auto.GetCertificate(hello)
	}

	return nil, ErrNoCert
}

func (rtr *Router) loadConfig() error {
	if logger := rtr.log.Load(); logger != nil {
		logger.(*slog.Logger).Info("reloading config", "fname", rtr.opts.ConfigFname)
	}
	var cfg config.Toplevel
	if err := hclsimple.DecodeFile(rtr.opts.ConfigFname, nil, &cfg); err != nil {
		return err
	}

	if err := cfg.Valid(); err != nil {
		return err
	}

	if err := rtr.setConfig(cfg); err != nil {
		return err
	}

	if logger := rtr.log.Load(); logger != nil {
		logger.(*slog.Logger).Info("done reloading config", "domains", len(cfg.Domains))
	}

	return nil
}

func (rtr *Router) backgroundReloadConfig(ctx context.Context) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			if err := rtr.loadConfig(); err != nil {
				slog.Error("can't reload config", "fname", rtr.opts.ConfigFname, "err", err)
			}
		}
	}
}

func NewRouter(c config.Toplevel, logLevel string) (*Router, error) {
	baseLog := logging.InitSlog(logLevel)
	result := &Router{
		routes:   map[string]http.Handler{},
		baseSlog: baseLog,
	}
	result.accessLog.Store((*lumberjack.Logger)(nil))
	result.log.Store(baseLog)

	if err := result.setConfig(c); err != nil {
		return nil, err
	}

	return result, nil
}

func (rtr *Router) HandleHTTP(ctx context.Context, ln net.Listener) error {
	srv := http.Server{
		Handler: rtr.httpHandler(),
	}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	return srv.Serve(ln)
}

func (rtr *Router) httpHandler() http.Handler {
	rtr.lock.RLock()
	auto := rtr.autoMgr
	redirectCode := rtr.autocertRedirectCode
	rtr.lock.RUnlock()

	if auto == nil {
		return rtr
	}

	mux := http.NewServeMux()
	// Serve ACME HTTP-01 challenges
	mux.Handle("/.well-known/acme-challenge/", auto.HTTPHandler(nil))
	// Redirect everything else to HTTPS preserving host and path
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u := *r.URL
		u.Scheme = "https"
		u.Host = r.Host
		http.Redirect(w, r, u.String(), redirectCode)
	})
	return mux
}

func (rtr *Router) HandleHTTPS(ctx context.Context, ln net.Listener) error {
	tc := &tls.Config{
		GetCertificate: rtr.GetCertificate,
	}

	srv := &http.Server{
		Handler:   rtr,
		TLSConfig: tc,
	}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	fingerprint.ApplyTLSFingerprinter(srv)

	return srv.ServeTLS(ln, "", "")
}

func (rtr *Router) ListenAndServeMetrics(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("(metrics) can't bind to tcp %s: %w", addr, err)
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/readyz", readyz)
	mux.HandleFunc("/healthz", healthz)

	if logger := rtr.log.Load(); logger != nil {
		logger.(*slog.Logger).Info("listening", "for", "metrics", "bind", addr)
	}

	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	return srv.Serve(ln)
}

func (rtr *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var host = r.Host

	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(host)
	}

	var h http.Handler
	var ok bool

	ja4hFP := ja4h.JA4H(r)

	if logger := rtr.log.Load(); logger != nil {
		logger.(*slog.Logger).Debug("got request", "method", r.Method, "host", host, "path", r.URL.Path)
	}

	rtr.lock.RLock()
	h, ok = rtr.routes[host]
	rtr.lock.RUnlock()

	if !ok {
		unresolvedRequests.Inc()
		http.NotFound(w, r) // TODO(Xe): brand this
		return
	}

	r.Header.Set("X-Http-Ja4h-Fingerprint", ja4hFP)

	if fp := fingerprint.GetTLSFingerprint(r); fp != nil {
		if ja3n := fp.JA3N(); ja3n != nil {
			r.Header.Set("X-Tls-Ja3n-Fingerprint", ja3n.String())
		}
		if ja4 := fp.JA4(); ja4 != nil {
			r.Header.Set("X-Tls-Ja4-Fingerprint", ja4.String())
		}
	}

	if tcpFP := fingerprint.GetTCPFingerprint(r); tcpFP != nil {
		r.Header.Set("X-Tcp-Ja4t-Fingerprint", tcpFP.String())
	}

	m := httpsnoop.CaptureMetrics(h, w, r)

	requestsPerDomain.WithLabelValues(host, r.Method, fmt.Sprint(m.Code)).Inc()
	responseTime.WithLabelValues(host).Observe(float64(m.Duration.Milliseconds()))

	if logger := rtr.log.Load(); logger != nil {
		logger.(*slog.Logger).Debug("request completed", "host", host, "method", r.Method, "response_code", m.Code, "duration_ms", m.Duration.Milliseconds())
	}

	if accessLog := rtr.accessLog.Load(); accessLog != nil {
		if logger, ok := accessLog.(*lumberjack.Logger); ok && logger != nil {
			logging.LogHTTPRequest(logger, r, m.Code, m.Written, m.Duration)
		}
	}
}
