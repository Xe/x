package main

import (
	"context"
	"flag"
	"net/url"
	"os"
	"time"

	"github.com/Xe/ln"
	_ "github.com/joho/godotenv/autoload"
)

var (
	every = flag.Duration("every", 10*time.Minute, "how often this binary is being run")
	do    = flag.Int("do", 1, "do this number of scale checks, staggered by -every")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	ctx = ln.WithF(ctx, ln.F{"at": "main"})

	pool, err := NewRedisPoolFromURL(os.Getenv("REDIS_URL"))
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	_ = pool

	for _, a := range flag.Args() {
		ln.Log(ctx, ln.F{"url": a})
		u, err := url.Parse(a)
		if err != nil {
			ln.FatalErr(ctx, err)
		}

		err = Check(ctx, u)
		if err != nil {
			ln.FatalErr(ctx, err)
		}
	}
}
