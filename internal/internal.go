// Package internal centralizes a lot of other boring configuration and startup logic into a common place.
package internal

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/posener/complete"
	"go4.org/legal"
	"within.website/confyg/flagconfyg"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal/flagenv"
	"within.website/x/internal/manpage"
	"within.website/x/web/useragent"

	// Debug routes
	"net/http"
	_ "net/http/pprof"

	// Older projects use .env files, shim in compatibility
	_ "github.com/joho/godotenv/autoload"
)

var (
	licenseShow = flag.Bool("license", false, "show software licenses?")
	config      = flag.String("config", "", "configuration file, if set (see flagconfyg(4))")
	writeConfig = flag.String("write-config", "", "if set, write flags to this file by name/path")
	manpageGen  = flag.Bool("manpage", false, "generate a manpage template?")
)

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
	if val := *writeConfig; val != "" {
		ln.Log(ctx, ln.Info("writing flags to file, remember to remove write-config"), ln.F{"fname": val})
		data := flagconfyg.Dump(flag.CommandLine)
		err := ioutil.WriteFile(val, data, 0644)
		if err != nil {
			ln.FatalErr(ctx, err)
		}
		os.Exit(0)
	}

	if *config != "" {
		ln.Log(ctx, ln.Info("loading config"), ln.F{"path": *config})

		flagconfyg.CmdParse(*config)
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

func init() {
	http.DefaultTransport = useragent.Transport("within.website-x", "https://within.website/.x.botinfo", http.DefaultTransport)
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
