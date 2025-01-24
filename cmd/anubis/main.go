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
	"log"
	"log/slog"
	"math"
	mrand "math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"within.website/x"
	"within.website/x/internal"
	"within.website/x/xess"
)

var (
	bind                = flag.String("bind", ":8923", "TCP port to bind HTTP to")
	challengeDifficulty = flag.Int("difficulty", 5, "difficulty of the challenge")
	metricsBind         = flag.String("metrics-bind", ":9090", "TCP port to bind metrics to")
	robotsTxt           = flag.Bool("serve-robots-txt", false, "serve a robots.txt file that disallows all robots")
	target              = flag.String("target", "http://localhost:3923", "target to reverse proxy to")

	//go:embed static
	static embed.FS

	bypasses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anubis_bypasses",
		Help: "The total number of requests that bypassed challenge validation",
	})

	challengesIssued = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anubis_challenges_issued",
		Help: "The total number of challenges issued",
	})

	challengesValidated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "anubis_challenges_validated",
		Help: "The total number of challenges validated",
	})

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

func main() {
	internal.HandleStartup()

	s, err := New(*target)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.Handle(staticPath, internal.UnchangingCache(http.StripPrefix(staticPath, http.FileServerFS(static))))
	xess.Mount(mux)

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
		"Accept-Language=%s,X-Real-IP=%s,User-Agent=%s,WeekTime=%s,Fingerprint=%x",
		r.Header.Get("Accept-Language"),
		r.Header.Get("X-Real-Ip"),
		r.UserAgent(),
		time.Now().UTC().Round(24*7*time.Hour).Format(time.RFC3339),
		fp,
	)
	result, _ := sha256sum(data)
	return result
}

func New(target string) (*Server, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	return &Server{
		rp:   rp,
		priv: priv,
		pub:  pub,
	}, nil
}

type Server struct {
	rp   *httputil.ReverseProxy
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

func (s *Server) maybeReverseProxy(w http.ResponseWriter, r *http.Request) {
	switch {
	case !strings.Contains(r.UserAgent(), "Mozilla"):
		bypasses.Inc()
		slog.Debug("non-browser user agent")
		s.rp.ServeHTTP(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/.well-known/"):
		bypasses.Inc()
		slog.Debug("well-known path")
		s.rp.ServeHTTP(w, r)
		return
	case strings.HasSuffix(r.URL.Path, ".rss") || strings.HasSuffix(r.URL.Path, ".xml") || strings.HasSuffix(r.URL.Path, ".atom"):
		bypasses.Inc()
		slog.Debug("rss path")
		s.rp.ServeHTTP(w, r)
		return
	case r.URL.Path == "/favicon.ico":
		bypasses.Inc()
		slog.Debug("favicon path")
		s.rp.ServeHTTP(w, r)
		return
	case r.URL.Path == "/robots.txt":
		bypasses.Inc()
		slog.Debug("robots.txt path")
		s.rp.ServeHTTP(w, r)
		return
	}

	ckie, err := r.Cookie(cookieName)
	if err != nil {
		slog.Debug("cookie not found", "path", r.URL.Path)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	if err := ckie.Valid(); err != nil {
		slog.Debug("cookie is invalid", "err", err)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	if time.Now().After(ckie.Expires) && !ckie.Expires.IsZero() {
		slog.Debug("cookie expired", "path", r.URL.Path)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	token, err := jwt.ParseWithClaims(ckie.Value, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.pub, nil
	})

	if !token.Valid {
		slog.Debug("invalid token", "path", r.URL.Path)
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	claims := token.Claims.(jwt.MapClaims)

	exp, ok := claims["exp"].(float64)
	if !ok {
		slog.Debug("exp is not int64", "ok", ok, "typeof(exp)", fmt.Sprintf("%T", exp))
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	if exp := time.Unix(int64(exp), 0); time.Now().After(exp) {
		slog.Debug("token has expired", "exp", exp.Format(time.RFC3339))
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

	if token.Valid && randomJitter() {
		slog.Debug("cookie is not enrolled into secondary screening")
		s.rp.ServeHTTP(w, r)
		return
	}

	if claims["challenge"] != s.challengeFor(r) {
		slog.Debug("invalid challenge", "path", r.URL.Path)
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
		slog.Error("failed to calculate sha256sum", "path", r.URL.Path, "err", err)
		clearCookie(w)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if subtle.ConstantTimeCompare([]byte(claims["response"].(string)), []byte(calculated)) != 1 {
		slog.Debug("invalid response", "path", r.URL.Path)
		failedValidations.Inc()
		clearCookie(w)
		s.renderIndex(w, r)
		return
	}

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

	json.NewEncoder(w).Encode(struct {
		Challenge  string `json:"challenge"`
		Difficulty int    `json:"difficulty"`
	}{
		Challenge:  challenge,
		Difficulty: difficulty,
	})
	slog.Debug("made challenge", "challenge", challenge, "difficulty", difficulty)
	challengesIssued.Inc()
}

func (s *Server) passChallenge(w http.ResponseWriter, r *http.Request) {
	nonceStr := r.FormValue("nonce")
	if nonceStr == "" {
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("missing nonce")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	elapsedTimeStr := r.FormValue("elapsedTime")
	if elapsedTimeStr == "" {
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("missing elapsedTime")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	elapsedTime, err := strconv.ParseFloat(elapsedTimeStr, 64)
	if err != nil {
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("invalid elapsedTime")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	difficultyStr := r.FormValue("difficulty")
	if difficultyStr == "" {
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("missing difficulty")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	difficulty, err := strconv.Atoi(difficultyStr)
	if err != nil {
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("invalid difficulty")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	slog.Info("challenge took", "elapsedTime", elapsedTime)
	timeTaken.Observe(elapsedTime)

	response := r.FormValue("response")
	redir := r.FormValue("redir")

	challenge := s.challengeFor(r)

	nonce, err := strconv.Atoi(nonceStr)
	if err != nil {
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("invalid nonce")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	calcString := fmt.Sprintf("%s%d", challenge, nonce)
	calculated, err := sha256sum(calcString)
	if err != nil {
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("failed to calculate sha256sum")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	if subtle.ConstantTimeCompare([]byte(response), []byte(calculated)) != 1 {
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("invalid response")), templ.WithStatus(http.StatusForbidden)).ServeHTTP(w, r)
		failedValidations.Inc()
		return
	}

	// compare the leading zeroes
	if !strings.HasPrefix(response, strings.Repeat("0", difficulty)) {
		clearCookie(w)
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
		slog.Error("failed to sign JWT", "err", err)
		clearCookie(w)
		templ.Handler(base("Oh noes!", errorPage("failed to sign JWT")), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tokenString,
		Expires:  time.Now().Add(24 * 7 * time.Hour),
		SameSite: http.SameSiteDefaultMode,
		Path:     "/",
	})

	challengesValidated.Inc()
	http.Redirect(w, r, redir, http.StatusFound)
}

func (s *Server) testError(w http.ResponseWriter, r *http.Request) {
	err := r.FormValue("err")
	templ.Handler(base("Oh noes!", errorPage(err)), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
}

func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    cookieName,
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
		MaxAge:  -1,
	})
}

func randomJitter() bool {
	return mrand.Intn(100) > 10
}
