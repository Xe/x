//+build ignore

// Makes the docker image xena/xperimental.
package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"

	"within.website/x/internal"
	"within.website/x/internal/yeet"
)

var (
	dontPush = flag.Bool("dont-push", false, "if set, don't push docker images")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tag := "xena/xperimental"
	yeet.DockerBuild(ctx, yeet.WD, tag)

	resTag := yeet.DockerTag(ctx, "xena", "xperimental", tag)
	otherResTag := yeet.DockerTag(ctx, "docker.pkg.github.com/xe/x", "xperimental", tag)

	gitTag, err := yeet.GitTag(ctx)
	if err != nil {
		log.Fatal(err)
	}

	dnsdTag := "xena/dnsd:" + gitTag

	yeet.DockerBuild(ctx, filepath.Join(yeet.WD, "cmd", "dnsd"), dnsdTag, "--build-arg", "X_VERSION="+gitTag)
	dnsdGithubTag := yeet.DockerTag(ctx, "docker.pkg.github.com/xe/x", "dnsd", dnsdTag)

	if !*dontPush {
		yeet.ShouldWork(ctx, nil, yeet.WD, "docker", "push", resTag)
		yeet.ShouldWork(ctx, nil, yeet.WD, "docker", "push", otherResTag)
		yeet.ShouldWork(ctx, nil, yeet.WD, "docker", "push", dnsdGithubTag)
	}

	log.Printf("xperimental:\t%s", otherResTag)
	log.Printf("dnsd:\t%s", dnsdGithubTag)
}
