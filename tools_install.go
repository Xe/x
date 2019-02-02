// +build ignore

package main

import (
	"context"
	"os"
	"time"

	"github.com/Xe/x/internal/yeet"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	env := append(os.Environ(), []string{"GO111MODULE=on"}...)

	for _, tool := range []string{"github.com/robertkrimen/godocdown/godocdown", "github.com/go-bindata/go-bindata/go-bindata"} {
		yeet.ShouldWork(ctx, env, yeet.WD, "go", "install", tool)
	}
}
