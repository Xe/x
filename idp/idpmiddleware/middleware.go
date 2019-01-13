package idpmiddleware

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/kr/pretty"
	"github.com/pborman/uuid"
	"within.website/ln"
	"within.website/ln/opname"
)

// Protect protects a given URL behind your given idp(1) server.
func Protect(idpServer, me, selfURL string) func(next http.Handler) http.Handler {
	lock := sync.Mutex{}
	codes := map[string]string{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = opname.With(ctx, "idpmiddleware.Protect.Handler")
			if r.URL.Path == "/auth/challenge" {
				v := r.URL.Query()
				ctx = ln.WithF(ctx, ln.F{"as": me, "state": v.Get("state"), "code": v.Get("code")})
				ln.Log(ctx, ln.Info("login"))
				lock.Lock()
				pretty.Println(codes)
				if cd := v.Get("state"); codes[cd] == cd {
					ln.Log(ctx, ln.Info("setting cookie"))
					http.SetCookie(w, &http.Cookie{
						Name:     "auth",
						Value:    me,
						HttpOnly: true,
						Expires:  time.Now().Add(900 * time.Hour),
						Path:     "/",
						SameSite: http.SameSiteLaxMode,
					})

					http.Error(w, "success", http.StatusOK)
				}
				lock.Unlock()
				return
			}

			cookie, err := r.Cookie("auth")
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
				v.Set("redirect_uri", selfURL+"auth/challenge")
				v.Set("state", code)
				v.Set("response_type", "id")
				u.RawQuery = v.Encode()

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
