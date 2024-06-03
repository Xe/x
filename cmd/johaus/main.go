// Command johaus wraps the lojban parser camxes and presents the results over HTTP.
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"

	"tulpa.dev/cadey/jvozba"
	"within.website/johaus/parser"
	_ "within.website/johaus/parser/camxes"
	"within.website/johaus/pretty"
	"within.website/x/internal"
)

var (
	port    = flag.String("port", "9001", "TCP port to bind on for HTTP")
	dialect = flag.String("dialect", "camxes", "Lojban dialect to use")
)

func main() {
	internal.HandleStartup()

	log.Printf("Listening on http://0.0.0.0:%s", *port)

	http.HandleFunc("/tree", tree)
	http.HandleFunc("/braces", braces)
	http.HandleFunc("/lujvo", lujvo)

	http.ListenAndServe(":"+*port, nil)
}

func lujvo(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	jvo, err := jvozba.Jvozba(string(data))
	if err != nil {
		http.Error(w, "can't read: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("X-Content-Language", "jbo")
	http.Error(w, jvo, http.StatusOK)
}

func braces(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't parse: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	tree, err := parser.Parse(*dialect, string(data))
	if err != nil {
		http.Error(w, "can't parse: "+err.Error(), http.StatusBadRequest)
		return
	}

	parser.RemoveMorphology(tree)
	parser.AddElidedTerminators(tree)
	parser.RemoveSpace(tree)
	parser.CollapseLists(tree)

	pretty.Braces(w, tree)
}

func tree(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't parse: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	tree, err := parser.Parse(*dialect, string(data))
	if err != nil {
		http.Error(w, "can't parse: "+err.Error(), http.StatusBadRequest)
		return
	}

	parser.RemoveMorphology(tree)
	parser.AddElidedTerminators(tree)
	parser.RemoveSpace(tree)
	parser.CollapseLists(tree)

	json.NewEncoder(w).Encode(tree)
}
