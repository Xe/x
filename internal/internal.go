package internal

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Xe/x/internal/flagenv"
	"github.com/Xe/x/internal/manpage"
	"go4.org/legal"
	"within.website/confyg/flagconfyg"
	"within.website/ln"
	"within.website/ln/opname"

	// Debug routes
	_ "net/http/pprof"

	// Older projects use .env files, shim in compatibility
	_ "github.com/joho/godotenv/autoload"

	// User agent init hook
	_ "github.com/Xe/x/web"
)

var (
	licenseShow = flag.Bool("license", false, "show software licenses?")
	config      = flag.String("config", "", "configuration file, if set (see flagconfyg(4))")
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

	ctx := opname.With(context.Background(), "internal.HandleStartup")
	if *config != "" {
		ln.Log(ctx, ln.Info("loading config"), ln.F{"path": *config})

		flagconfyg.CmdParse(*config)
	}
	flag.Parse()
	flagenv.Parse()

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
