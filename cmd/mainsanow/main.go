// Command mainsanow shows the current time in ma insa.
package main

import (
	"log"
	"time"

	"within.website/x/internal"
	"within.website/x/internal/mainsa"
)

func main() {
	internal.HandleStartup()

	tn, err := mainsa.At(time.Now())
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s", tn)
}
