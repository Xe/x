//+build ignore

// Builds and deploys the application to minipaas.
package main

import (
	"context"
	"log"
	"os"

	"github.com/Xe/x/internal"
	"github.com/Xe/x/internal/greedo"
	"github.com/Xe/x/internal/minipaas"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := append(os.Environ(), []string{"CGO_ENABLED=0", "GOOS=linux"}...)
	internal.ShouldWork(ctx, env, internal.WD, "vgo", "build", "-o=sona-pi-toki-pona")
	internal.ShouldWork(ctx, env, internal.WD, "appsluggr", "-worker=sona-pi-toki-pona")
	fin, err := os.Open("slug.tar.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	fname := "sona-pi-toki-pona-" + internal.DateTag + ".tar.gz"
	pubURL, err := greedo.CopyFile(fname, fin)
	if err != nil {
		log.Fatal(err)
	}

	err = minipaas.Exec("tar:from sona-pi-toki-pona " + pubURL)
	if err != nil {
		log.Fatal(err)
	}
}
