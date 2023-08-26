package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (s *Sanguisuga) UntrackTV(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var show Show
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&show)
	if err != nil {
		slog.Error("can't read request body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	s.db.Data.TVWatch = remove(s.db.Data.TVWatch, func(s Show) bool {
		return s.Title == show.Title
	})

	slog.Info("no longer tracking TV show", "show", show)

	if err := s.db.Save(); err != nil {
		slog.Error("can't save database", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Sanguisuga) TrackTV(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var show Show
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&show)
	if err != nil {
		slog.Error("can't read request body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	s.db.Data.TVWatch = append(s.db.Data.TVWatch, show)
	if err := s.db.Save(); err != nil {
		slog.Error("can't save database", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(s.db.Data.TVWatch)
}

func (s *Sanguisuga) ListTVSnatches(w http.ResponseWriter, r *http.Request) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	json.NewEncoder(w).Encode(s.db.Data.TVSnatches)
}

func (s *Sanguisuga) ListTV(w http.ResponseWriter, r *http.Request) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	json.NewEncoder(w).Encode(s.db.Data.TVWatch)
}
