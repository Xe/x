//+build ignore

// Builds and deploys the application to minipaas.
package main

import (
	"context"
	"log"
	"os"

	"github.com/Xe/x/internal/greedo"
	"github.com/Xe/x/internal/minipaas"
	"github.com/Xe/x/internal/yeet"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := append(os.Environ(), []string{"CGO_ENABLED=0", "GOOS=linux"}...)
	yeet.ShouldWork(ctx, env, yeet.WD, "vgo", "build", "-o=web")
	yeet.ShouldWork(ctx, env, yeet.WD, "appsluggr", "-web=web")
	fin, err := os.Open("slug.tar.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	fname := "la-banbixsicns-" + yeet.DateTag + ".tar.gz"
	pubURL, err := greedo.CopyFile(fname, fin)
	if err != nil {
		log.Fatal(err)
	}

	err = minipaas.Exec("tar:from la-banbixsicns " + pubURL)
	if err != nil {
		log.Fatal(err)
	}
}
