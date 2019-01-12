package internal

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/Xe/x/tools/license/licenses"
	"go4.org/legal"
	"within.website/confyg/flagconfyg"
	"within.website/ln"
	"within.website/ln/opname"
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
		json.NewEncoder(w).Encode(legal.Licenses())
	})
}

// HandleLicense is a wrapper for commands that use HandleLicense.
func HandleLicense() { HandleStartup() }

// HandleStartup optionally shows all software licenses or other things.
func HandleStartup() {
	ctx := opname.With(context.Background(), "internal.HandleStartup")
	if *licenseShow {
		fmt.Printf("Licenses for %v\n", os.Args)

		for _, li := range legal.Licenses() {
			fmt.Println(li)
			fmt.Println()
		}

		os.Exit(0)
	}

	if *config != "" {
		ln.Log(ctx, ln.Info("loading config"), ln.F{"path": *config})

		flagconfyg.CmdParse(*config)
	}
}
