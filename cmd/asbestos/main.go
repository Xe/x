package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	_ "github.com/mattn/go-sqlite3"
	"within.website/x/htmx"
	"within.website/x/internal"
	"within.website/x/xess"
)

//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

var (
	bind  = flag.String("bind", ":8069", "http port to bind on")
	dbURL = flag.String("database-url", "", "path to sqlite database")
)

func main() {
	internal.HandleStartup()

	slog.Info("opening database", "database_url", *dbURL)
	db, err := sql.Open("sqlite3", *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := &Server{db: db}

	mux := http.NewServeMux()
	xess.Mount(mux)
	htmx.Mount(mux)

	mux.Handle("/{$}", templ.Handler(
		Layout("Asbestos", Index()),
	))

	mux.HandleFunc("POST /search", s.searchHTMXPage)

	fmt.Printf("http://localhost%s\n", *bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}

type Server struct {
	db *sql.DB
}

type Post struct {
	ID         int
	Text       string
	CreatedAt  string
	Author     string
	URI        string
	HasImages  bool
	ReplyTo    sql.NullString
	BlueskyURL string
}

func (s *Server) getPostsByAuthor(author string) ([]Post, error) {
	rows, err := s.db.Query("SELECT id, text, created_at, author, uri, has_images, reply_to, bluesky_url FROM posts WHERE author = ?", author)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Text, &post.CreatedAt, &post.Author, &post.URI, &post.HasImages, &post.ReplyTo, &post.BlueskyURL); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (s *Server) searchHTMXPage(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("search")

	posts, err := s.getPostsByAuthor(query)
	if err != nil {
		slog.Error("can't find posts", "author", query, "err", err)
		templ.Handler(Error(err.Error())).ServeHTTP(w, r)
		return
	}

	if len(posts) == 0 {
		templ.Handler(allClear()).ServeHTTP(w, r)
		return
	}

	templ.Handler(searchPage(query, posts)).ServeHTTPStreamed(w, r)
}
