//+build ignore

// Makes the docker image xena/xperimental.
package main

import (
	"context"
	"log"
	"path/filepath"

	"within.website/x/internal/yeet"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tag := "xena/xperimental"
	yeet.ShouldWork(ctx, nil, yeet.WD, "docker", "build", "-t", tag, ".")

	resTag, err := yeet.DockerTag(ctx, "xena", "xperimental", tag)
	if err != nil {
		log.Fatal(err)
	}
	gitTag, err := yeet.GitTag(ctx)
	if err != nil {
		log.Fatal(err)
	}

	dnsdTag := "xena/dnsd:" + gitTag

	yeet.ShouldWork(ctx, nil, filepath.Join(yeet.WD, "cmd", "dnsd"), "docker", "build", "-t", dnsdTag, "--build-arg", "X_VERSION="+gitTag, ".")

	yeet.ShouldWork(ctx, nil, yeet.WD, "docker", "push", resTag)
	yeet.ShouldWork(ctx, nil, yeet.WD, "docker", "push", dnsdTag)

	log.Printf("use %s", resTag)
}
