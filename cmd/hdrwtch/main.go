package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/mymmrac/telego"
	"github.com/robfig/cron/v3"
	"within.website/x/htmx"
	"within.website/x/internal"
)

//go:generate tailwindcss --input styles.css --output static/css/styles.css
//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

var (
	botToken     = flag.String("bot-token", "", "Telegram bot token")
	botUsername  = flag.String("bot-username", "", "Telegram bot username")
	cookieSecret = flag.String("cookie-secret", "", "Secret key for cookie store")
	dbLoc        = flag.String("database-loc", "./var/hdrwtch.db", "Database location")
	domain       = flag.String("domain", "shiroko-wsl.shark-harmonic.ts.net", "Domain to use for user agent")
	port         = flag.String("port", "8080", "Port to listen on")
	region       = flag.String("fly-region", "yow-dev", "Region of this instance")

	//dbURL        = flag.String("database-url", "", "Database URL")

	//go:embed static
	staticFS embed.FS
)

func main() {
	internal.HandleStartup()

	if *cookieSecret == "" {
		fmt.Println("cookie-secret is required")
		fmt.Printf("%x\n", securecookie.GenerateRandomKey(32))
		os.Exit(1)
	}

	bot, err := telego.NewBot(*botToken)
	if err != nil {
		log.Fatal(err)
	}

	dao, err := New(*dbLoc)
	if err != nil {
		log.Fatal(err)
	}

	s := &Server{
		store: sessions.NewCookieStore([]byte(*cookieSecret)),
		dao:   dao,
		tg:    bot,
	}

	if err := s.importDocs(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	htmx.Mount(mux)

	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	mux.HandleFunc("/{$}", s.index)
	mux.HandleFunc("/login", s.loginHandler)
	mux.HandleFunc("/login/callback", s.loginCallbackHandler)
	mux.HandleFunc("/logout", s.logoutHandler)
	mux.HandleFunc("/docs/{slug...}", s.docsHandler)

	// authed routes
	mux.Handle("/user", s.loggedIn(s.userHandler))

	// probe management
	mux.Handle("GET /probe", s.loggedIn(s.probeList))
	mux.Handle("POST /probe", s.loggedIn(s.probeCreate))
	mux.Handle("GET /probe/{id}", s.loggedIn(s.probeGet))
	mux.Handle("GET /probe/{id}/edit", s.loggedIn(s.probeEdit))
	mux.Handle("PUT /probe/{id}", s.loggedIn(s.probeUpdate))
	mux.Handle("DELETE /probe/{id}", s.loggedIn(s.probeDelete))
	mux.Handle("GET /probe/{id}/run/{result_id}", s.loggedIn(s.probeRunGet))

	mux.Handle("/", internal.UnchangingCache(
		templ.Handler(
			base("Not Found", nil, anonNavBar(true), notFoundPage()),
			templ.WithStatus(http.StatusNotFound),
		),
	))

	// test routes
	mux.HandleFunc("GET /test/curr", func(w http.ResponseWriter, r *http.Request) {
		val := time.Now().Format(http.TimeFormat)
		w.Header().Set("Last-Modified", val)
		fmt.Fprintln(w, val)
	})
	mux.HandleFunc("GET /test/constant", func(w http.ResponseWriter, r *http.Request) {
		val := "Mon, 02 Jan 2006 15:04:05 GMT"
		w.Header().Set("Last-Modified", val)
		fmt.Fprintln(w, val)
	})

	c := cron.New()
	if *region == "yow-dev" {
		c.AddFunc("@every 1m", s.cron)
		slog.Info("running in dev mode", "cron-frequency", "1m")
	} else {
		c.AddFunc("@every 15m", s.cron)
	}
	go c.Start()

	slog.Info("listening", "on", "http://localhost:"+*port)

	log.Fatal(http.ListenAndServe(":"+*port, mux))
}

type Server struct {
	store *sessions.CookieStore
	dao   *DAO
	tg    *telego.Bot
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	var navbar templ.Component

	tu, ok := s.getTelegramUserData(r)
	if ok {
		navbar = authedNavBar(tu)
	} else {
		navbar = anonNavBar(true)
	}

	templ.Handler(base("hdrwtch", nil, navbar, homePage())).ServeHTTP(w, r)
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := s.store.Get(r, "telegram-session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/user", http.StatusFound)
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.getTelegramUserData(r); ok {
		http.Redirect(w, r, "/user", http.StatusFound)
		return
	}

	u := r.URL.JoinPath("/callback")

	templ.Handler(base("Login", nil, anonNavBar(false), loginPage(u.String()))).ServeHTTP(w, r)
}

func (s *Server) loginCallbackHandler(w http.ResponseWriter, r *http.Request) {
	authData := make(map[string]string)
	for key, values := range r.URL.Query() {
		authData[key] = values[0]
	}

	user, err := checkTelegramAuthorization(*botToken, authData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	err = s.saveTelegramUserData(w, r, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.dao.UpsertUser(r.Context(), user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/user", http.StatusFound)
}

func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) {
	userData, ok := s.getTelegramUserData(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	probeCount, err := s.dao.CountProbes(r.Context(), userData.ID)
	if err != nil {
		slog.Error("failed to count probes", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templ.Handler(
		base(
			"User Info",
			nil,
			authedNavBar(userData),
			userPage(userData, probeCount),
		),
	).ServeHTTP(w, r)
}
