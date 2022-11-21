// Package internal centralizes a lot of other boring configuration and startup logic into a common place.
package internal

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/posener/complete"
	"go4.org/legal"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal/confyg/flagconfyg"
	"within.website/x/internal/flagenv"
	"within.website/x/internal/manpage"

	// Debug routes
	_ "expvar"
	_ "net/http/pprof"

	// Older projects use .env files, shim in compatibility
	_ "github.com/joho/godotenv/autoload"
)

var (
	licenseShow = flag.Bool("license", false, "show software licenses?")
	config      = flag.String("config", configFileLocation(), "configuration file, if set (see flagconfyg(4))")
	manpageGen  = flag.Bool("manpage", false, "generate a manpage template?")
)

func configFileLocation() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		ln.Error(context.Background(), err, ln.Debug("can't read config dir"))
		return ""
	}

	dir = filepath.Join(dir, "within.website", "x")
	os.MkdirAll(dir, 0700)

	return filepath.Join(dir, filepath.Base(os.Args[0])+".config")
}

// HandleStartup optionally shows all software licenses or other things.
// This always loads from the following configuration sources in the following
// order:
//
//     - command line flags (to get -config)
//     - environment variables
//     - configuration file (if -config is set)
//     - command line flags
//
// This is done this way to ensure that command line flags always are the deciding
// factor as an escape hatch.
func HandleStartup() {
	flag.Parse()
	flagenv.Parse()

	ctx := opname.With(context.Background(), "internal.HandleStartup")

	if *config != "" {
		ln.Log(ctx, ln.Info("loading config"), ln.F{"path": *config})

		flagconfyg.CmdParse(ctx, *config)
	}
	flag.Parse()

	if *licenseShow {
		fmt.Printf("Licenses for %v\n", os.Args)

		for _, li := range legal.Licenses() {
			fmt.Println(li)
			fmt.Println()
		}

		os.Exit(0)
	}

	if *manpageGen {
		manpage.Spew()
	}
}

func HandleCompletion(args complete.Predictor, subcommands complete.Commands) {
	cmd := complete.Command{
		Flags: map[string]complete.Predictor{},
		Sub:   subcommands,
		Args:  args,
	}

	flag.CommandLine.VisitAll(func(fl *flag.Flag) {
		cmd.Flags["-"+fl.Name] = complete.PredictAnything

		if fl.DefValue == "true" || fl.DefValue == "false" {
			cmd.Flags["-"+fl.Name] = complete.PredictNothing
		}
	})

	if complete.New(filepath.Base(os.Args[0]), cmd).Run() {
		os.Exit(0)
	}
}
