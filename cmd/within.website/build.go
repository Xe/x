//+build ignore

// Builds and deploys the application to minipaas.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/vfsgen"
	"within.website/x/internal"
	"within.website/x/internal/kahless"
	"within.website/x/internal/minipaas"
	"within.website/x/internal/yeet"
)

var (
	genVFS = flag.Bool("gen-vfs", true, "if true, generate VFS")
	deploy = flag.Bool("deploy", true, "if true, deploy to minipaas via kahless")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *genVFS {
		err := vfsgen.Generate(http.Dir("./static"), vfsgen.Options{
			PackageName:  "main",
			BuildTags:    "!dev",
			VariableName: "assets",
		})
		if err != nil {
			log.Fatalln(err)
		}
	}

	if *deploy {
		env := append(os.Environ(), []string{"CGO_ENABLED=0", "GOOS=linux"}...)
		yeet.ShouldWork(ctx, env, yeet.WD, "go", "build", "-o=web")
		yeet.ShouldWork(ctx, env, yeet.WD, "appsluggr", "-web=web")
		fin, err := os.Open("slug.tar.gz")
		if err != nil {
			log.Fatal(err)
		}
		defer fin.Close()

		fname := "within.website-" + yeet.DateTag + ".tar.gz"
		pubURL, err := kahless.CopyFile(fname, fin)
		if err != nil {
			log.Fatal(err)
		}

		err = minipaas.Exec("tar:from within.website " + pubURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}
