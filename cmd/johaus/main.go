package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"within.website/x/internal"
	"within.website/johaus/parser"
	_ "within.website/johaus/parser/camxes-beta"
	"within.website/johaus/pretty"
)

var (
	port    = flag.String("port", "9001", "TCP port to bind on for HTTP")
	dialect = flag.String("dialect", "camxes-beta", "Lojban dialect to use")
)

func main() {
	internal.HandleStartup()

	log.Printf("Listening on http://0.0.0.0:%s", *port)

	http.DefaultServeMux.HandleFunc("/tree", tree)
	http.DefaultServeMux.HandleFunc("/braces", braces)

	http.ListenAndServe(":"+*port, nil)
}

func braces(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't parse: "+err.Error(), http.StatusBadRequest)
		return
	}

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
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't parse: "+err.Error(), http.StatusBadRequest)
		return
	}

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
