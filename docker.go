//+build ignore

// Makes the docker image xena/xperimental.
package main

import (
	"context"
	"log"

	"github.com/Xe/x/internal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tag := "xena/xperimental:" + internal.DateTag

	internal.ShouldWork(ctx, nil, internal.WD, "docker", "build", "-t", tag, ".")
	internal.ShouldWork(ctx, nil, internal.WD, "docker", "push", tag)

	log.Printf("use %s", tag)
}
