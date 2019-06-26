package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	maxBytes = flag.Int64("max-playground-bytes", 75, "how many bytes of data should users be allowed to post to the playground?")
)

func doHTTP() error {
	http.HandleFunc("/api/playground", runPlayground)

	return http.ListenAndServe(":"+*port, nil)
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
		http.Error(w, "too many bytes sent", http.StatusBadRequest)
		return
	}

	comp, err := compile(string(data))
	if err != nil {
		http.Error(w, fmt.Sprintf("compilation error: %v", err), http.StatusBadRequest)
		return
	}

	er, err := run(comp.Binary)
	if err != nil {
		http.Error(w, fmt.Sprintf("runtime error: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(struct {
		Program *CompiledProgram `json:"prog"`
		Results *ExecResult      `json:"res"`
	}{
		Program: comp,
		Results: er,
	})
}
