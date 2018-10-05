package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/Xe/x/internal"
	"github.com/Xe/x/internal/minipaas"
)

func main() {
	flag.Parse()
	internal.HandleLicense()

	client, err := minipaas.Dial()
	if err != nil {
		log.Fatal(err)
	}

	sess, err := client.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr

	err = sess.Run(strings.Join(flag.Args(), " "))
	if err != nil {
		log.Fatal(err)
	}
}
