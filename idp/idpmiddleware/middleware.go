package idpmiddleware

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pborman/uuid"
	"within.website/ln"
	"within.website/ln/opname"
)

// hash is a simple wrapper around the MD5 algorithm implementation in the
// Go standard library. It takes in data and a salt and returns the hashed
// representation.
func hash(data string, salt string) string {
	output := md5.Sum([]byte(data + salt))
	return fmt.Sprintf("%x", output)
}

func verify(ctx context.Context, idpServer, state, code string) *http.Request {
	u, err := url.Parse(idpServer)
	if err != nil {
		panic(err)
	}

	u.Path = "/auth"
	q := u.Query()
	q.Set("code", code)
	q.Set("state", state)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "application/json")
	req = req.WithContext(ctx)

	return req
}

func validate(resp *http.Response) (string, error) {
	result := struct {
		Me string `json:"me"`
	}{}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("wanted status 200, got: %d", resp.StatusCode)
	}

	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	return result.Me, nil
}

// XeProtect sets defaults for Xe to use.
func XeProtect(selfURL string) func(next http.Handler) http.Handler {
	return Protect("https://idp.christine.website", "https://christine.website/", selfURL)
}

// Protect protects a given URL behind your given idp(1) server.
func Protect(idpServer, me, selfURL string) func(next http.Handler) http.Handler {
	lock := sync.Mutex{}
	codes := map[string]string{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = opname.With(ctx, "idpmiddleware.Protect.Handler")
			if r.URL.Path == "/.within/x/idpmiddleware/challenge" {
				v := r.URL.Query()
				ctx = ln.WithF(ctx, ln.F{"as": me, "state": v.Get("state"), "code": v.Get("code")})
				ln.Log(ctx, ln.Info("login"))
				lock.Lock()
				defer lock.Unlock()
				if cd := v.Get("state"); codes[cd] == cd {
					ctx = opname.With(ctx, "verify")
					resp, err := http.DefaultClient.Do(verify(ctx, idpServer, v.Get("state"), v.Get("code")))
					if err != nil {
						ln.Error(ctx, err)
						http.Error(w, "nope", http.StatusInternalServerError)
						return
					}

					got, err := validate(resp)
					if err != nil {
						ln.Error(ctx, err)
						http.Error(w, "not valid", http.StatusInternalServerError)
						return
					}

					if me != got {
						ln.Error(ctx, errors.New("hacking attempt"))
						http.Error(w, "...", http.StatusNotAcceptable)
						return
					}

					ln.Log(ctx, ln.Info("setting cookie"))
					http.SetCookie(w, &http.Cookie{
						Name:     "within-x-idpmiddleware",
						Value:    hash(me, idpServer),
						HttpOnly: true,
						Expires:  time.Now().Add(900 * time.Hour),
						Path:     "/",
						SameSite: http.SameSiteLaxMode,
					})
					delete(codes, cd)

					w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
					w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
					http.Redirect(w, r, selfURL, http.StatusTemporaryRedirect)
				}

				http.Error(w, "Programmer error, maybe you have multiple instances of the IDP middleware?", http.StatusInternalServerError)
				return
			}

			cookie, err := r.Cookie("within-x-idpmiddleware")
			if err != nil || cookie.Value != hash(me, idpServer) {
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
				v.Set("redirect_uri", selfURL+".within/x/idpmiddleware/challenge")
				v.Set("state", code)
				v.Set("response_type", "id")
				u.RawQuery = v.Encode()

				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
				w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
				http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
