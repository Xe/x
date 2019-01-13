package idpmiddleware

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pborman/uuid"
)

// Protect protects a given URL behind your given idp(1) server.
func Protect(idpServer, me, selfURL string) func(next http.Handler) http.Handler {
	lock := sync.Mutex{}
	codes := map[string]string{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/auth/challenge" {
				v := r.URL.Query()
				lock.Lock()
				defer lock.Unlock()
				if cd := v.Get("code"); codes[cd] == cd {
					http.SetCookie(w, &http.Cookie{
						Name:     "idp",
						Value:    me,
						HttpOnly: true,
						Expires:  time.Now().Add(900 * time.Hour),
					})

					http.Redirect(w, r, selfURL, http.StatusTemporaryRedirect)
					return
				}
			}

			cookie, err := r.Cookie("idp")
			if err != nil {
				u, err := url.Parse(idpServer)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				code := uuid.New()
				lock.Lock()
				codes[code] = code
				lock.Unlock()

				u.Path = "/auth"
				v := url.Values{}
				v.Set("me", me)
				v.Set("client_id", selfURL)
				v.Set("redirect_uri", selfURL+"/auth/challenge")
				v.Set("state", code)
				v.Set("response_type", "id")

				http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
				return
			}

			if cookie.Value != me {
				http.Error(w, "wrong identity", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
