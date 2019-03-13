package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"sync"
	"text/template"
	"time"

	"github.com/Xe/x/i18n"
	"github.com/Xe/x/idp/idpmiddleware"
	"github.com/Xe/x/internal"
	"github.com/pborman/uuid"
	"github.com/xlzd/gotp"
	"within.website/ln"
	"within.website/ln/ex"
)

var (
	domain      = flag.String("domain", "idp.christine.website", "domain to be hosted from")
	otpSecret   = flag.String("otp-secret", "", "OTP secret")
	port        = flag.String("port", "5484", "TCP port to listen on for HTTP")
	owner       = flag.String("owner", "https://christine.website/", "the me=that is required")
	secretGen   = flag.Int("secret-gen", 0, "generate a secret of len if set")
	defaultLang = flag.String("default-language", "en_US", "default language if none is set")
	gitRev      = flag.String("git-rev", "", "git revision of runtime (used for dokku detection)")
)

func main() {
	internal.HandleStartup()

	if *secretGen != 0 {
		log.Fatal(gotp.RandomSecret(*secretGen))
	}

	translationPath := "./translations"
	if *gitRev != "" {
		translationPath = "/app/translations"
	}

	l := i18n.New(*defaultLang, translationPath)

	i := &idp{
		t:         gotp.NewDefaultTOTP(*otpSecret),
		l:         l,
		bearer2me: map[string]string{},
	}

	log.Println(i.t.ProvisioningUri(*domain, *domain))

	def := idpmiddleware.XeProtect("https://" + *domain + "/")(http.DefaultServeMux)
	mux := http.NewServeMux()
	mux.HandleFunc("/lang", func(w http.ResponseWriter, r *http.Request) {
		locales := i18n.GetLocales(r)

		json.NewEncoder(w).Encode(locales)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tr := i.l.TranslationsForRequest(r)
		fm := template.FuncMap{
			"T": tr.Value,
		}

		t, err := template.New("root").Funcs(fm).Parse(rootPageTemplate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = t.Execute(w, nil)
		if err != nil {
			log.Printf("%v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	mux.HandleFunc("/auth", i.auth)
	mux.HandleFunc("/challenge", i.challenge)
	mux.Handle("/.within/", def)
	mux.Handle("/debug/", def)
	http.ListenAndServe(":"+*port, ex.HTTPLog(mux))
}

type idp struct {
	t *gotp.TOTP

	l *i18n.L
	sync.Mutex
	bearer2me map[string]string
}

// auth implements https://indieweb.org/authorization-endpoint#Open_Source_Authorization_Endpoints
func (i *idp) auth(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("idp: form error in /auth: %v", err)
		return
	}

	var (
		code, me, state, responseType, redirectURI, clientID string
	)
	for k, v := range r.Form {
		switch k {
		case "state":
			state = v[0]
		case "me":
			me = v[0]
		case "response_type":
			responseType = v[0]
		case "redirect_uri":
			redirectURI = v[0]
		case "client_id":
			clientID = v[0]
		case "code":
			code = v[0]
		}
	}

	if code != "" {
		i.Lock()
		person := i.bearer2me[code]
		delete(i.bearer2me, code)
		i.Unlock()

		ctx := r.Context()
		ln.Log(ctx, ln.F{"state": state, "code": code, "accept": r.Header.Get("Accept"), "person": person})

		w.Header().Set("Content-Type", r.Header.Get("Accept"))
		switch r.Header.Get("Accept") {
		case "application/x-www-form-urlencoded":
			v := url.Values{}
			v.Set("me", person)

			http.Error(w, v.Encode(), http.StatusOK)
		case "application/json":
			json.NewEncoder(w).Encode(struct {
				Me string `json:"me"`
			}{
				Me: person,
			})
		}

		return
	}

	if me != *owner {
		http.Error(w, "Not allowed", http.StatusUnauthorized)
		return
	}

	tr := i.l.TranslationsForRequest(r)
	fm := template.FuncMap{
		"T": tr.Value,
	}

	t, err := template.New("auth").Funcs(fm).Parse(authPageTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, struct {
		ClientID, State, Me, ResponseType, RedirectURI string
	}{
		ClientID:     clientID,
		State:        state,
		Me:           me,
		ResponseType: responseType,
		RedirectURI:  redirectURI,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (i *idp) challenge(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("idp: form error in /auth: %v", err)
		return
	}

	var (
		code, me, state, redirectURI string
	)
	for k, v := range r.Form {
		switch k {
		case "code":
			code = v[0]
		case "state":
			state = v[0]
		case "me":
			me = v[0]
		case "redirect_uri":
			redirectURI = v[0]
		}
	}

	nowCode := i.t.Now()
	if code != nowCode {
		http.Error(w, "Not allowed", http.StatusUnauthorized)
		return
	}

	bearerToken := uuid.New()
	i.Lock()
	i.bearer2me[bearerToken] = me
	i.Unlock()

	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "url error", http.StatusBadRequest)
		return
	}

	time.Sleep(125 * time.Millisecond)

	q := u.Query()
	q.Set("state", state)
	q.Set("code", bearerToken)
	u.RawQuery = q.Encode()

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}

const rootPageTemplate = `<html>
<head>
<link rel="stylesheet" href="https://unpkg.com/chota@0.5.2/dist/chota.min.css">
<title>{{ T "errors.forbidden" }}</title>
<meta name=viewport content="width=400">
<style>
:root {
  --color-primary: #da1d50; /* brand color */
  --grid-maxWidth: 40rem;
}
</style>
</head>
<body id="top">
<div class="container">
<div class="card">
  <header>
    <h4>{{ T "errors.forbidden" }}</h4>
  </header>

  <p>{{ T "prose.index" }}</p>
</div>
</div>
</body>
</html>`

const authPageTemplate = `<html>
<head>
<link rel="stylesheet" href="https://unpkg.com/chota@0.5.2/dist/chota.min.css">
<title>{{ T "auth_title" }}</title>
<meta name=viewport content="width=400">
<style>
:root {
  --color-primary: #da1d50; /* brand color */
  --grid-maxWidth: 40rem;
}
</style>
</head>
<body id="top">
<div class="container">
<div class="card">
  <header>
    <h4>{{ T "prose.auth" .ClientID .Me }}</h4>
  </header>
  <p><form action="/challenge" method="GET">
  {{ T "code" }} <br>
  <input type="text" name="code" value="" autofocus><br><br>
  <input type="hidden" name="me" value="{{ .Me }}">
  <input type="hidden" name="state" value="{{ .State }}">
  <input type="hidden" name="client_id" value="{{ .ClientID }}">
  <input type="hidden" name="response_type" value="{{ .ResponseType }}">
  <input type="hidden" name="redirect_uri" value="{{ .RedirectURI }}">
  <input class="button primary" type="submit" value="{{ T "submit_button" }}">
</form></p>
</div>
</div>
</body>
</html>`
