package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Xe/x/web/ghstat"
)

var (
	messageFlag = flag.Bool("message", false, "show last message?")

	// TODO: implement
	//shellFlag = flag.Bool("s", false, "show as shell prompt artifact")
)

func main() {
	flag.Parse()

	if *messageFlag {
		req := ghstat.LastMessage()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		var m ghstat.Message
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&m)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Last message:\n")
		fmt.Printf("Status:  %s\n", m.Status)
		fmt.Printf("Message: %s\n", m.Body)
		fmt.Printf("Time:    %s\n", m.CreatedOn)

		return
	}

	req := ghstat.LastStatus()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	var s ghstat.Status
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&s)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Status: %s (%s)\n", s.Status, s.LastUpdated)

	if s.Status != "good" {
		os.Exit(1)
	}
}
