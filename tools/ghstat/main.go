package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

var (
	messageFlag = flag.Bool("message", false, "show last message?")

	// TODO: implement
	//shellFlag = flag.Bool("s", false, "show as shell prompt artifact")
)

func main() {
	flag.Parse()

	if *messageFlag {
		m, err := getMessage()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Last message:\n")
		fmt.Printf("Status:  %s\n", m.Status)
		fmt.Printf("Message: %s\n", m.Body)

		t, err := time.Parse(time.RFC3339, m.CreatedOn)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Time:    %s\n", t.Format(time.ANSIC))
	} else {
		s, err := getStatus()
		if err != nil {
			log.Fatal(err)
		}

		t, err := time.Parse(time.RFC3339, s.LastUpdated)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Status: %s (%s)\n", s.Status, t.Format(time.ANSIC))

		if s.Status != "good" {
			os.Exit(1)
		}
	}
}
