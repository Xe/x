//+build ignore

// Builds and deploys the application to minipaas.
package main

import (
	"context"
	"log"
	"os"

	"within.website/x/internal"
	"within.website/x/internal/kahless"
	"within.website/x/internal/minipaas"
	"within.website/x/internal/yeet"
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := append(os.Environ(), []string{"CGO_ENABLED=0", "GOOS=linux"}...)
	yeet.ShouldWork(ctx, env, yeet.WD, "go", "build", "-o=web")
	yeet.ShouldWork(ctx, env, yeet.WD, "appsluggr", "-web=web")
	fin, err := os.Open("slug.tar.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	fname := "idp-" + yeet.DateTag + ".tar.gz"
	pubURL, err := kahless.CopyFile(fname, fin)
	if err != nil {
		log.Fatal(err)
	}

	err = minipaas.Exec("tar:from idp " + pubURL)
	if err != nil {
		log.Fatal(err)
	}
}
