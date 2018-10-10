//+build ignore

// Makes the docker image xena/xperimental.
package main

import (
	"context"
	"log"

	"github.com/Xe/x/internal/yeet"
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

	yeet.ShouldWork(ctx, nil, yeet.WD, "docker", "push", resTag)

	log.Printf("use %s", resTag)
}
