// Command whoisfront is a simple CGI wrapper to switchcounter.science. This is used in some internal tooling.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cgi"
	"os"

	"within.website/x/internal"
)

var (
	miTokenPath = flag.String("mi-token-path", "", "Mi token path")
)

func main() {
	internal.HandleStartup()

	err := cgi.Serve(http.HandlerFunc(handle))
	if err != nil {
		log.Fatal(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest(http.MethodGet, "https://mi.within.website/api/switches/current/text", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := os.ReadFile(*miTokenPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Add("Authorization", string(token))
	req.Header.Add("Accept", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("bad status code: %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	io.Copy(w, resp.Body)
}
