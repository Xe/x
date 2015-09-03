package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/tv42/httpunix"
)

func main() {
	u := &httpunix.Transport{
		DialTimeout:           100 * time.Millisecond,
		RequestTimeout:        2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
	}
	u.RegisterLocation("docker", "/var/run/docker.sock")

	var client = http.Client{
		Transport: u,
	}

	resp, err := client.Get("http+unix://docker/v1.12/containers/e1778874f60b5a4ff18a122606bfe2ba45ba41e2ba71b27ce8de1e2dd403aabd/stats?stream=0")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	reply := &Stats{}

	err = json.Unmarshal(data, reply)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%#v", reply)
}
