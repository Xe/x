package internal

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/Xe/x/internal/flagenv"
	"github.com/Xe/x/internal/manpage"
	"github.com/Xe/x/tools/license/licenses"
	"go4.org/legal"
	"within.website/confyg/flagconfyg"
	"within.website/ln"
	"within.website/ln/opname"

	// Debug routes
	_ "net/http/pprof"

	// Older projects use .env files, shim in compatibility
	_ "github.com/joho/godotenv/autoload"
)

var (
	licenseShow = flag.Bool("license", false, "show software licenses?")
	config      = flag.String("config", "", "configuration file, if set (see flagconfyg(4))")
	manpageGen  = flag.Bool("manpage", false, "generate a manpage template?")
)

func init() {
	legal.RegisterLicense(licenses.CC0License)
	legal.RegisterLicense(licenses.SQLiteBlessing)

	http.HandleFunc("/.within/licenses", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Licenses for this program: %s\n", os.Args[0])

		for _, li := range legal.Licenses() {
			fmt.Fprintln(w, li)
			fmt.Fprintln(w)
		}

		fmt.Fprintln(w, "Be well, Creator.")
	})
}

// HandleLicense is a wrapper for commands that use HandleLicense.
func HandleLicense() { HandleStartup() }

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
	HandleConfig(ctx)
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

// HandleConfig handles the config file parsing from -config
func HandleConfig(ctx context.Context) {
	if *config != "" {
		ln.Log(ctx, ln.Info("loading config"), ln.F{"path": *config})

		flagconfyg.CmdParse(*config)
	}
	flag.Parse()
}
