// Command whoisfront is a simple CGI wrapper to switchcounter.science. This is used in some internal tooling.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"

	"within.website/x/internal"
)

var (
	miToken = flag.String("mi-token", "", "Mi token")
)

func main() {
	internal.HandleStartup()

	err := cgi.Serve(http.HandlerFunc(handle))
	if err != nil {
		log.Fatal(err)
	}
}

func miSwitch(to string) error {
	req, err := http.NewRequest(http.MethodGet, "https://mi.within.website/switches/switch", bytes.NewBuffer([]byte(to)))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", *miToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wanted %d, got: %s", http.StatusOK, resp.Status)
	}
	return nil
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		front, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		defer r.Body.Close()

		err = miSwitch(string(front))
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, string(front))
		return
	}

	req, err := http.NewRequest(http.MethodGet, "https://mi.within.website/switches/current", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", *miToken)
	req.Header.Add("Accept", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Panicf("bad status code: %d", resp.StatusCode)
	}

	w.Header().Set("Content-Type", "text/plain")
	io.Copy(w, resp.Body)
}
