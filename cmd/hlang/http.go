package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/rs/cors"
	"within.website/x/cmd/hlang/h"
	"within.website/x/cmd/hlang/run"
	"within.website/x/xess"
)

//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

var (
	maxBytes = flag.Int64("max-playground-bytes", 75, "how many bytes of data should users be allowed to post to the playground?")
)

func doHTTP() error {
	http.Handle("/{$}", templ.Handler(xess.Base("The h Programming Language", nil, navbar(), homePage(), footer())))
	http.Handle("/docs", templ.Handler(xess.Base("Documentation", nil, navbar(), docsPage(), footer())))
	http.Handle("/faq", templ.Handler(xess.Base("FAQ", nil, navbar(), faqPage(), footer())))
	http.Handle("/play", templ.Handler(xess.Base("Playground", nil, navbar(), playgroundPage(), footer())))
	http.HandleFunc("/api/playground", runPlayground)

	http.Handle("/grammar/", http.StripPrefix("/grammar/", http.FileServer(http.FS(h.Grammar))))

	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: cors.Default().Handler(http.DefaultServeMux),
	}

	if *sockpath != "" {
		os.RemoveAll(*sockpath)

		l, err := net.Listen("unix", *sockpath)
		if err != nil {
			return fmt.Errorf("can't listen on %s: %v", *sockpath, err)
		}
		defer l.Close()

		return srv.Serve(l)
	} else {
		return srv.ListenAndServe()
	}
}

func httpError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(struct {
		Error string `json:"err"`
	}{
		Error: err.Error(),
	})
}

func runPlayground(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	rc := http.MaxBytesReader(w, r.Body, *maxBytes)
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		httpError(w, err, http.StatusBadGateway)
		return
	}

	comp, err := compile(string(data))
	if err != nil {
		httpError(w, fmt.Errorf("compliation error: %v", err), http.StatusBadRequest)
		return
	}

	er, err := run.Run(comp.Binary)
	if err != nil {
		httpError(w, fmt.Errorf("runtime error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Program *CompiledProgram `json:"prog"`
		Results *run.ExecResult  `json:"res"`
	}{
		Program: comp,
		Results: er,
	})
}

const usage = `Usage of hlang:
-config string
      configuration file, if set (see flagconfyg(4))
-koan
      if true, print the h koan and then exit
-license
      show software licenses?
-manpage
      generate a manpage template?
-max-playground-bytes int
      how many bytes of data should users be allowed to post to the playground? (default 75)
-o string
      if specified, write the webassembly binary created by -p here
-p string
      h program to compile/run
-port string
      HTTP port to listen on
-slog-level string
      log level (default "INFO")
-sockpath string
      Unix domain socket to listen on
-v    if true, print the version of h and then exit`
