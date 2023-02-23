//go:build ignore

// Builds and deploys the application to fly.io.
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
	yeet.ShouldWork(ctx, env, yeet.WD, "nix", "build", ".#docker.xedn")
	yeet.DockerLoadResult(ctx, "./result")
	yeet.DockerPush(ctx, "registry.fly.io/xedn:latest")
	yeet.ShouldWork(ctx, env, yeet.WD, "flyctl", "deploy", "--now")
}
