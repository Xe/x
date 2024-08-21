package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"gorm.io/gorm"
	"within.website/x/htmx"
)

type Probe struct {
	gorm.Model
	UserID       string
	Name         string
	URL          string
	LastResultID uint
	LastResult   ProbeResult
}

type ProbeResult struct {
	gorm.Model
	ProbeID      uint
	Success      bool
	LastModified string
	StatusCode   int
	Region       string
	Remark       string
	Duration     time.Duration
}

func (s *Server) probeList(w http.ResponseWriter, r *http.Request) {
	tu, ok := s.getTelegramUserData(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var probes []Probe

	if err := s.dao.db.Find(&probes).Joins("LastResult").Where("id = ?", tu.ID).Error; err != nil {
		slog.Error("failed to get probes", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templ.Handler(
		base(
			"Probes",
			nil,
			authedNavBar(tu),
			probeListPage(probes),
		),
	).ServeHTTP(w, r)
}

func (s *Server) probeCreate(w http.ResponseWriter, r *http.Request) {
	tu, ok := s.getTelegramUserData(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	newProbe := &Probe{
		UserID: tu.ID,
		Name:   r.FormValue("name"),
		URL:    r.FormValue("url"),
	}

	if err := s.dao.db.Create(newProbe).Error; err != nil {
		slog.Error("failed to create probe", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var probes []Probe

	if err := s.dao.db.Find(&probes).Joins("LastResult").Where("id = ?", tu.ID).Error; err != nil {
		slog.Error("failed to get probes", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templ.Handler(
		probeListPage(probes),
	).ServeHTTP(w, r)
}

func (s *Server) probeEdit(w http.ResponseWriter, r *http.Request) {
	tu, ok := getTelegramUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	probe, err := s.dao.GetProbe(r.Context(), r.PathValue("id"), tu.ID)
	if err != nil {
		slog.Error("failed to get probe", "path", r.URL.Path, "err", err)
		http.Error(w, "no probe data", http.StatusUnauthorized)
		return
	}

	templ.Handler(
		probeEdit(*probe),
	).ServeHTTP(w, r)
}

func (s *Server) probeGet(w http.ResponseWriter, r *http.Request) {
	tu, ok := getTelegramUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	probe, err := s.dao.GetProbe(r.Context(), r.PathValue("id"), tu.ID)
	if err != nil {
		slog.Error("failed to get probe", "path", r.URL.Path, "err", err)
		http.Error(w, "no probe data", http.StatusUnauthorized)
		return
	}

	if htmx.Is(r) {
		templ.Handler(
			probeRow(*probe),
		).ServeHTTP(w, r)
	} else {
		var results []ProbeResult

		if err := s.dao.db.Find(&results).Where("probe_id = ?", probe.ID).Order("created_at DESC").Limit(15).Error; err != nil {
			slog.Error("failed to get probe results", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		templ.Handler(
			base(probe.Name, nil, authedNavBar(tu), probePage(*probe, results)),
		).ServeHTTP(w, r)
	}
}

func (s *Server) probeUpdate(w http.ResponseWriter, r *http.Request) {
	tu, ok := getTelegramUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	probe, err := s.dao.GetProbe(r.Context(), r.PathValue("id"), tu.ID)
	if err != nil {
		slog.Error("failed to get probe", "path", r.URL.Path, "err", err)
		http.Error(w, "no probe data", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	probe.Name = r.FormValue("name")
	probe.URL = r.FormValue("url")

	if err := s.dao.db.Save(probe).Error; err != nil {
		slog.Error("failed to update probe", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templ.Handler(
		probeRow(*probe),
	).ServeHTTP(w, r)
}

func (s *Server) probeDelete(w http.ResponseWriter, r *http.Request) {
	tu, ok := getTelegramUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	probe, err := s.dao.GetProbe(r.Context(), r.PathValue("id"), tu.ID)
	if err != nil {
		slog.Error("failed to get probe", "path", r.URL.Path, "err", err)
		http.Error(w, "no probe data", http.StatusUnauthorized)
		return
	}

	if err := s.dao.db.Delete(probe).Error; err != nil {
		slog.Error("failed to delete probe", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/probes", http.StatusFound)
}
