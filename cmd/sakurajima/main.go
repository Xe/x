package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"within.website/x"
	"within.website/x/cmd/sakurajima/internal/entrypoint"
	"within.website/x/cmd/sakurajima/internal/logging"
	"within.website/x/internal/flagenv"
)

var (
	configFname = flag.String("config", "./sakurajima.hcl", "Configuration file (HCL), see docs")
	slogLevel   = flag.String("slog-level", "INFO", "logging level (see https://pkg.go.dev/log/slog#hdr-Levels)")
	versionFlag = flag.Bool("version", false, "if true, show version information then quit")
)

func main() {
	flagenv.Parse()
	flag.Parse()

	if *versionFlag {
		fmt.Println("Sakurajima", x.Version)
		return
	}

	slog.SetDefault(logging.InitSlog(*slogLevel))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := entrypoint.Main(ctx, entrypoint.Options{
		ConfigFname: *configFname,
		LogLevel:    *slogLevel,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
