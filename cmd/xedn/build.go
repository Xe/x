//+build ignore

// Builds and deploys the application to minipaas.
package main

import (
	"context"
	"os"

	"within.website/x/internal"
	"within.website/x/internal/yeet"
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := append(os.Environ(), []string{"CGO_ENABLED=0", "GOOS=linux"}...)
	yeet.ShouldWork(ctx, env, yeet.WD, "go", "build", "-v", "-o=web")
	yeet.ShouldWork(ctx, env, yeet.WD, "appsluggr", "-web=web")
	os.Remove("web")
	yeet.ShouldWork(ctx, env, yeet.WD, "flyctl", "deploy", "--now")
}