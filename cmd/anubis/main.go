package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	mrand "math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"within.website/x"
	"within.website/x/cmd/anubis/internal/config"
	"within.website/x/cmd/anubis/internal/dnsbl"
	"within.website/x/internal"
	"within.website/x/xess"
)

var (
	bind                = flag.String("bind", ":8923", "TCP port to bind HTTP to")
	challengeDifficulty = flag.Int("difficulty", 5, "difficulty of the challenge")
	metricsBind         = flag.String("metrics-bind", ":9090", "TCP port to bind metrics to")
	robotsTxt           = flag.Bool("serve-robots-txt", false, "serve a robots.txt file that disallows all robots")
	policyFname         = flag.String("policy-fname", "", "full path to anubis policy document (defaults to a sensible built-in policy)")
	target              = flag.String("target", "http://localhost:3923", "target to reverse proxy to")
	healthcheck         = flag.Bool("healthcheck", false, "run a health check against Anubis")

	//go:embed static botPolicies.json
	static embed.FS

	challengesIssued = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anubis_challenges_issued",
		Help: "The total number of challenges issued",
	})

	challengesValidated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anubis_challenges_validated",
		Help: "The total number of challenges validated",
	})

	droneBLHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anubis_dronebl_hits",
		Help: "The total number of hits from DroneBL",
	}, []string{"status"})

	failedValidations = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anubis_failed_validations",
		Help: "The total number of failed validations",
	})

	timeTaken = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "anubis_time_taken",
		Help:    "The time taken for a browser to generate a response (milliseconds)",
		Buckets: prometheus.ExponentialBucketsRange(1, math.Pow(2, 18), 19),
	})
)

const (
	cookieName = "within.website-x-cmd-anubis-auth"
	staticPath = "/.within.website/x/cmd/anubis/"
)

//go:generate go run github.com/a-h/templ/cmd/templ@latest generate
//go:generate esbuild js/main.mjs --sourcemap --minify --bundle --outfile=static/js/main.mjs
//go:generate gzip -f -k static/js/main.mjs
//go:generate zstd -f -k --ultra -22 static/js/main.mjs
//go:generate brotli -fZk static/js/main.mjs

func doHealthCheck() error {
	resp, err := http.Get("http://localhost" + *metricsBind + "/metrics")
	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	internal.HandleStartup()

	if *healthcheck {
		if err := doHealthCheck(); err != nil {
			log.Fatal(err)
		}
		return
	}

	s, err := New(*target, *policyFname)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	xess.Mount(mux)

	mux.Handle(staticPath, internal.UnchangingCache(http.StripPrefix(staticPath, http.FileServerFS(static))))

	// mux.HandleFunc("GET /.within.website/x/cmd/anubis/static/js/main.mjs", serveMainJSWithBestEncoding)

	mux.HandleFunc("POST /.within.website/x/cmd/anubis/api/make-challenge", s.makeChallenge)
	mux.HandleFunc("GET /.within.website/x/cmd/anubis/api/pass-challenge", s.passChallenge)
	mux.HandleFunc("GET /.within.website/x/cmd/anubis/api/test-error", s.testError)

	if *robotsTxt {
		mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, static, "static/robots.txt")
		})

		mux.HandleFunc("/.well-known/robots.txt", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, static, "static/robots.txt")
		})
	}

	if *metricsBind != "" {
		go metricsServer()
	}

	mux.HandleFunc("/", s.maybeReverseProxy)

	slog.Info("listening", "url", "http://localhost"+*bind, "difficulty", *challengeDifficulty, "serveRobotsTXT", *robotsTxt, "target", *target, "version", x.Version)
	log.Fatal(http.ListenAndServe(*bind, mux))
}

func metricsServer() {
	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())
	slog.Debug("listening for metrics", "url", "http://localhost"+*metricsBind)
	log.Fatal(http.ListenAndServe(*metricsBind, nil))
}

func sha256sum(text string) (string, error) {
	hash := sha256.New()
	_, err := hash.Write([]byte(text))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *Server) challengeFor(r *http.Request) string {
	fp := sha256.Sum256(s.priv.Seed())

	data := fmt.Sprintf(
		"Accept-Language=%s,X-Real-IP=%s,User-Agent=%s,WeekTime=%s,Fingerprint=%x,Difficulty=%d",
		r.Header.Get("Accept-Language"),
		r.Header.Get("X-Real-Ip"),
		r.UserAgent(),
		time.Now().UTC().Round(24*7*time.Hour).Format(time.RFC3339),
		fp,
		*challengeDifficulty,
	)
	result, _ := sha256sum(data)
	return result
}

func New(target, policyFname string) (*Server, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	var fin io.ReadCloser

	if policyFname != "" {
		fin, err = os.Open(policyFname)
		if err != nil {
			return nil, fmt.Errorf("can't parse policy file %s: %w", policyFname, err)
		}
	} else {
		policyFname = "(static)/botPolicies.json"
		fin, err = static.Open("botPolicies.json")
		if err != nil {
			return nil, fmt.Errorf("[unexpected] can't parse builtin policy file %s: %w", policyFname, err)
		}
	}

	defer fin.Close()

	policy, err := parseConfig(fin, policyFname)
	if err != nil {
		return nil, err // parseConfig sets a fancy error for us
	}

	return &Server{
		rp:         rp,
		priv:       priv,
		pub:        pub,
		policy:     policy,
		dnsblCache: NewDecayMap[string, dnsbl.DroneBLResponse](),
	}, nil
}

type Server struct {
	rp         *httputil.ReverseProxy
	priv       ed25519.PrivateKey
	pub        ed25519.PublicKey
	policy     *ParsedConfig
	dnsblCache *DecayMap[string, dnsbl.DroneBLResponse]
}

func (s *Server) maybeReverseProxy(w http.ResponseWriter, r *http.Request) {
	cr := s.check(r)
	r.Header.Add("X-Anubis-Rule", cr.Name)
	r.Header.Add("X-Anubis-Action", string(cr.Rule))
	lg := slog.With(
		"check_result", cr,
		"user_agent", r.UserAgent(),
		"accept_language", r.Header.Get("Accept-Language"),
		"priority", r.Header.Get("Priority"),
		"x-forwarded-for",
		r.Header.Get("X-Forwarded-For"),
		"x-real-ip", r.Header.Get("X-Real-Ip"),
	)
	policyApplications.WithLabelValues(cr.Name, string(cr.Rule)).Add(1)

	ip := r.Header.Get("X-Real-Ip")

	if s.policy.DNSBL && ip != "" {
		resp, ok := s.dnsblCache.Get(ip)
		if !ok {
			lg.Debug("looking up ip in dnsbl")
			resp, err := dnsbl.Lookup(ip)
			if err != nil {
				lg.Error("can't look up ip in dnsbl", "err", err)
			}
			s.dnsblCache.Set(ip, resp, 24*time.Hour)
			droneBLHits.WithLabelValues(resp.String()).Inc()
		}

		if resp != dnsbl.AllGood {
			lg.Info("DNSBL hit", "status", resp.String())
			templ.Handler(base("Oh noes!", errorPage(fmt.Sprintf("DroneBL reported an entry: %s, see https://dronebl.org/lookup?ip=%s", resp.String(), ip))), templ.WithStatus(http.StatusOK)).ServeHTTP(w, r)
			return
		}
	}

	switch cr.Rule {
	case config.RuleAllow:
		lg.Debug("allowing traffic to origin (explicit)")
		s.rp.ServeHTTP(w, r)
		return
	case config.RuleDeny:
		clearCookie(w)
		lg.Info("explicit deny")
		templ.Handler(base("Oh noes!", errorPage("Access Denied")), templ.WithStatus(http.StatusOK)).ServeHTTP(w, r)
		return
	case config.RuleChallenge:
		lg.Debug("challenge requested")
	default:
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("Other internal server error (contact the admin)")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	ckie, err := r.Cookie(cookieName)
	if err != nil {
		lg.Debug("cookie not found", "path", r.URL.Path)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	if err := ckie.Valid(); err != nil {
		lg.Debug("cookie is invalid", "err", err)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	if time.Now().After(ckie.Expires) && !ckie.Expires.IsZero() {
		lg.Debug("cookie expired", "path", r.URL.Path)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	token, err := jwt.ParseWithClaims(ckie.Value, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.pub, nil
	})

	if !token.Valid {
		lg.Debug("invalid token", "path", r.URL.Path)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	claims := token.Claims.(jwt.MapClaims)

	exp, ok := claims["exp"].(float64)
	if !ok {
		lg.Debug("exp is not int64", "ok", ok, "typeof(exp)", fmt.Sprintf("%T", exp))
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	if exp := time.Unix(int64(exp), 0); time.Now().After(exp) {
		lg.Debug("token has expired", "exp", exp.Format(time.RFC3339))
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	if token.Valid && randomJitter() {
		r.Header.Add("X-Anubis-Status", "PASS-BRIEF")
		lg.Debug("cookie is not enrolled into secondary screening")
		s.rp.ServeHTTP(w, r)
		return
	}

	if claims["challenge"] != s.challengeFor(r) {
		lg.Debug("invalid challenge", "path", r.URL.Path)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	var nonce int

	if v, ok := claims["nonce"].(float64); ok {
		nonce = int(v)
	}

	calcString := fmt.Sprintf("%s%d", s.challengeFor(r), nonce)
	calculated, err := sha256sum(calcString)
	if err != nil {
		lg.Error("failed to calculate sha256sum", "path", r.URL.Path, "err", err)
		clearCookie(w)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if subtle.ConstantTimeCompare([]byte(claims["response"].(string)), []byte(calculated)) != 1 {
		lg.Debug("invalid response", "path", r.URL.Path)
		failedValidations.Inc()
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	slog.Debug("all checks passed")
	r.Header.Add("X-Anubis-Status", "PASS-FULL")
	s.rp.ServeHTTP(w, r)
}

func (s *Server) renderIndex(w http.ResponseWriter, r *http.Request) {
	templ.Handler(
		base("Making sure you're not a bot!", index()),
	).ServeHTTP(w, r)
}

func (s *Server) makeChallenge(w http.ResponseWriter, r *http.Request) {
	challenge := s.challengeFor(r)
	difficulty := *challengeDifficulty

	lg := slog.With("user_agent", r.UserAgent(), "accept_language", r.Header.Get("Accept-Language"), "priority", r.Header.Get("Priority"), "x-forwarded-for", r.Header.Get("X-Forwarded-For"), "x-real-ip", r.Header.Get("X-Real-Ip"))

	json.NewEncoder(w).Encode(struct {
		Challenge  string `json:"challenge"`
		Difficulty int    `json:"difficulty"`
	}{
		Challenge:  challenge,
		Difficulty: difficulty,
	})
	lg.Debug("made challenge", "challenge", challenge, "difficulty", difficulty)
	challengesIssued.Inc()
}

func (s *Server) passChallenge(w http.ResponseWriter, r *http.Request) {
	lg := slog.With("user_agent", r.UserAgent(), "accept_language", r.Header.Get("Accept-Language"), "priority", r.Header.Get("Priority"), "x-forwarded-for", r.Header.Get("X-Forwarded-For"), "x-real-ip", r.Header.Get("X-Real-Ip"))

	nonceStr := r.FormValue("nonce")
	if nonceStr == "" {
		clearCookie(w)
		lg.Debug("no nonce")
		templ.Handler(base("Oh noes!", errorPage("missing nonce")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	elapsedTimeStr := r.FormValue("elapsedTime")
	if elapsedTimeStr == "" {
		clearCookie(w)
		lg.Debug("no elapsedTime")
		templ.Handler(base("Oh noes!", errorPage("missing elapsedTime")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	elapsedTime, err := strconv.ParseFloat(elapsedTimeStr, 64)
	if err != nil {
		clearCookie(w)
		lg.Debug("elapsedTime doesn't parse", "err", err)
		templ.Handler(base("Oh noes!", errorPage("invalid elapsedTime")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	lg.Info("challenge took", "elapsedTime", elapsedTime)
	timeTaken.Observe(elapsedTime)

	response := r.FormValue("response")
	redir := r.FormValue("redir")

	challenge := s.challengeFor(r)

	nonce, err := strconv.Atoi(nonceStr)
	if err != nil {
		clearCookie(w)
		lg.Debug("nonce doesn't parse", "err", err)
		templ.Handler(base("Oh noes!", errorPage("invalid nonce")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	calcString := fmt.Sprintf("%s%d", challenge, nonce)
	calculated, err := sha256sum(calcString)
	if err != nil {
		clearCookie(w)
		lg.Debug("can't parse shasum", "err", err)
		templ.Handler(base("Oh noes!", errorPage("failed to calculate sha256sum")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	if subtle.ConstantTimeCompare([]byte(response), []byte(calculated)) != 1 {
		clearCookie(w)
		lg.Debug("hash does not match", "got", response, "want", calculated)
		templ.Handler(base("Oh noes!", errorPage("invalid response")), templ.WithStatus(http.StatusForbidden)).ServeHTTP(w, r)
		failedValidations.Inc()
		return
	}

	// compare the leading zeroes
	if !strings.HasPrefix(response, strings.Repeat("0", *challengeDifficulty)) {
		clearCookie(w)
		lg.Debug("difficulty check failed", "response", response, "difficulty", *challengeDifficulty)
		templ.Handler(base("Oh noes!", errorPage("invalid response")), templ.WithStatus(http.StatusForbidden)).ServeHTTP(w, r)
		failedValidations.Inc()
		return
	}

	// generate JWT cookie
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"challenge": challenge,
		"nonce":     nonce,
		"response":  response,
		"iat":       time.Now().Unix(),
		"nbf":       time.Now().Add(-1 * time.Minute).Unix(),
		"exp":       time.Now().Add(24 * 7 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(s.priv)
	if err != nil {
		lg.Error("failed to sign JWT", "err", err)
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("failed to sign JWT")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tokenString,
		Expires:  time.Now().Add(24 * 7 * time.Hour),
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	challengesValidated.Inc()
	lg.Debug("challenge passed, redirecting to app")
	http.Redirect(w, r, redir, http.StatusFound)
}

func (s *Server) testError(w http.ResponseWriter, r *http.Request) {
	err := r.FormValue("err")
	templ.Handler(base("Oh noes!", errorPage(err)), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
}

func ohNoes(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("super fatal error", "err", err)
	templ.Handler(base("Oh noes!", errorPage("An internal server error happened")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
}

func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func randomJitter() bool {
	return mrand.Intn(100) > 10
}

func serveMainJSWithBestEncoding(w http.ResponseWriter, r *http.Request) {
	priorityList := []string{"zstd", "br", "gzip"}
	enc2ext := map[string]string{
		"zstd": "zst",
		"br":   "br",
		"gzip": "gz",
	}

	for _, enc := range priorityList {
		if strings.Contains(r.Header.Get("Accept-Encoding"), enc) {
			w.Header().Set("Content-Type", "text/javascript")
			w.Header().Set("Content-Encoding", enc)
			http.ServeFileFS(w, r, static, "static/js/main.mjs."+enc2ext[enc])
			return
		}
	}

	w.Header().Set("Content-Type", "text/javascript")
	http.ServeFileFS(w, r, static, "static/js/main.mjs")
}
