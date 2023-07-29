package internal

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"golang.org/x/exp/slog"
)

func init() {
	http.HandleFunc("/.within/debug/buildinfo", func(w http.ResponseWriter, r *http.Request) {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			slog.Error("can't read build info")
			http.Error(w, "no build info available", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(bi); err != nil {
			slog.Error("can't encode build info", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
