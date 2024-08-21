package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"gorm.io/gorm/clause"
	"within.website/x/htmx"
	"within.website/x/internal"
)

//go:generate tailwindcss --input styles.css --output static/css/styles.css --minify
//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

var (
	botToken     = flag.String("bot-token", "", "Telegram bot token")
	botUsername  = flag.String("bot-username", "", "Telegram bot username")
	cookieSecret = flag.String("cookie-secret", "", "Secret key for cookie store")
	dbURL        = flag.String("database-url", "", "Database URL")
	dbLoc        = flag.String("database-loc", "./var/hdrwtch.db", "Database location")
	port         = flag.String("port", "8080", "Port to listen on")

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

	dao, err := New(*dbLoc)
	if err != nil {
		log.Fatal(err)
	}

	s := &Server{
		store: sessions.NewCookieStore([]byte(*cookieSecret)),
		dao:   dao,
	}

	mux := http.NewServeMux()

	htmx.Mount(mux)

	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	mux.Handle("/{$}", templ.Handler(base("Home", nil, anonNavBar(true), homePage())))
	mux.HandleFunc("/login", s.loginHandler)
	mux.HandleFunc("/login/callback", s.loginCallbackHandler)
	mux.HandleFunc("/logout", s.logoutHandler)

	// authed routes
	mux.Handle("/user", s.loggedIn(s.userHandler))

	// probe management
	mux.Handle("GET /probe", s.loggedIn(s.probeList))
	mux.Handle("POST /probe", s.loggedIn(s.probeCreate))
	mux.Handle("GET /probe/{id}", s.loggedIn(s.probeGet))
	mux.Handle("GET /probe/{id}/edit", s.loggedIn(s.probeEdit))
	mux.Handle("PUT /probe/{id}", s.loggedIn(s.probeUpdate))
	mux.Handle("DELETE /probe/{id}", s.loggedIn(s.probeDelete))

	mux.Handle("/", templ.Handler(
		base("Not Found", nil, anonNavBar(true), notFoundPage()),
		templ.WithStatus(http.StatusNotFound),
	))

	slog.Info("listening", "on", "http://localhost:"+*port)

	log.Fatal(http.ListenAndServe(":"+*port, mux))
}

type Server struct {
	store *sessions.CookieStore
	dao   *DAO
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := s.store.Get(r, "telegram-session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/user", http.StatusFound)
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if userData, ok := s.getTelegramUserData(r); ok {
		slog.Info("user data", "ok", ok, "data", userData)

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

	if err := s.dao.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},                                                           // primary key
		DoUpdates: clause.AssignmentColumns([]string{"first_name", "last_name", "photo_url", "auth_date"}), // column needed to be updated
	}).Create(user).WithContext(r.Context()).Error; err != nil {
		slog.Error("failed to create user", "err", err, "user", user)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/user", http.StatusFound)
}

func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) {
	userData, ok := r.Context().Value(ctxKeyTelegramUser).(*TelegramUser)
	if !ok {
		http.Error(w, "no user data", http.StatusUnauthorized)
		return
	}

	templ.Handler(
		base(
			"User Info",
			nil,
			authedNavBar(userData),
			userPage(userData),
		),
	).ServeHTTP(w, r)
}
