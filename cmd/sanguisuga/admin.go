package main

import (
	"embed"
	"log/slog"
	"net/http"
)

var (
	//go:embed tmpl/*
	templates embed.FS

	//go:embed static
	static embed.FS
)

//go:generate tailwindcss --output static/styles.css --minify

func (s *Sanguisuga) AdminIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		s.tmpl.ExecuteTemplate(w, "404.tmpl", struct {
			Title string
		}{
			Title: "Not found: " + r.URL.Path,
		})

		return
	}
	s.tmpl.ExecuteTemplate(w, "index.html", struct {
		Title string
	}{
		Title: "sanguisuga",
	})
}

func (s *Sanguisuga) AdminAnimeList(w http.ResponseWriter, r *http.Request) {
	err := s.tmpl.ExecuteTemplate(w, "anime_index.html", struct {
		Title string
	}{
		Title: "Anime",
	})
	if err != nil {
		slog.Error("can't render template", "err", err)
	}
}

func (s *Sanguisuga) AdminTVList(w http.ResponseWriter, r *http.Request) {
	err := s.tmpl.ExecuteTemplate(w, "tv_index.html", struct {
		Title string
	}{
		Title: "TV",
	})
	if err != nil {
		slog.Error("can't render template", "err", err)
	}
}
