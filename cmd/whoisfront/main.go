// Command whoisfront is a simple CGI wrapper to switchcounter.science. This is used in some internal tooling.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"

	"within.website/x/internal"
	"within.website/x/web/switchcounter"
)

var (
	switchCounterURL = flag.String("switch-counter-url", "", "the webhook for switchcounter.science")
	miToken          = flag.String("mi-token", "", "Mi token")

	sc switchcounter.API
)

func main() {
	internal.HandleStartup()

	sc = switchcounter.NewHTTPClient(*switchCounterURL)

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
		req := sc.Switch(string(front))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		err = switchcounter.Validate(resp)
		if err != nil {
			panic(err)
		}

		err = miSwitch(string(front))
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, string(front))
		return
	}

	req := sc.Status()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	err = switchcounter.Validate(resp)
	if err != nil {
		panic(err)
	}
	var st switchcounter.Status
	err = json.NewDecoder(resp.Body).Decode(&st)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, st.Front)
}
