/*
This is an example application to demonstrate querying the user info endpoint.
*/
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"within.website/x/internal"
)

var (
	clientID     = flag.String("oauth2-client-id", "", "OAuth2 client ID")
	clientSecret = flag.String("oauth2-client-secret", "", "OAuth2 client secret")
	idpURL       = flag.String("oauth2-idp-url", "https://idp.xeserv.us", "OAuth2 IDP URL")
	redirectURL  = flag.String("oauth2-redirect-url", "http://127.0.0.1:5556/auth/callback", "OAuth2 redirect URL")
)

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCallbackCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, c)
}

func main() {
	internal.HandleStartup()

	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, *idpURL)
	if err != nil {
		log.Fatal(err)
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: *clientID})

	config := oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  *redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		state, err := randString(16)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		setCallbackCookie(w, r, "state", state)

		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	})

	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		state, err := r.Cookie("state")
		if err != nil {
			http.Error(w, "state not found", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("state") != state.Value {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var extraClaims any = nil

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if ok {
			idToken, err := verifier.Verify(ctx, rawIDToken)
			if err != nil {
				http.Error(w, "Failed to verify id_token: "+err.Error(), http.StatusInternalServerError)
				return
			}

			var claims struct {
				Subject           string   `json:"sub"`
				Email             string   `json:"email"`
				Verified          bool     `json:"email_verified"`
				Name              string   `json:"name"`
				PreferredUsername string   `json:"preferred_username"`
				Groups            []string `json:"groups"`
				Picture           string   `json:"picture"`
			}

			if err := idToken.Claims(&claims); err != nil {
				http.Error(w, "Failed to unmarshal token claims: "+err.Error(), http.StatusInternalServerError)
			}

			extraClaims = claims
		}

		userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
		if err != nil {
			http.Error(w, "Failed to get userinfo: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := struct {
			OAuth2Token *oauth2.Token
			UserInfo    *oidc.UserInfo
			ExtraClaims any
		}{oauth2Token, userInfo, extraClaims}
		data, err := json.MarshalIndent(resp, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(data)
	})

	log.Printf("listening on http://%s/", "127.0.0.1:5556")
	log.Fatal(http.ListenAndServe("127.0.0.1:5556", nil))
}
