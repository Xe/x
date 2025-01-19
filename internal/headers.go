package internal

import (
	"net/http"

	"within.website/x"
)

func UnchangingCache(h http.Handler) http.Handler {
	if x.Version == "devel" {
		return h
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		h.ServeHTTP(w, r)
	})
}
