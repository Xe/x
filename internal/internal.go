package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Xe/x/tools/license/licenses"
	"go4.org/legal"
)

var (
	licenseShow = flag.Bool("license", false, "show software license?")
)

func init() {
	legal.RegisterLicense(licenses.CC0License)
	legal.RegisterLicense(licenses.SQLiteBlessing)

	http.HandleFunc("/.within/licenses", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(legal.Licenses())
	})
}

// HandleLicense optionally shows all software licenses.
func HandleLicense() {
	if *licenseShow {
		log.Printf("Licenses for %v", os.Args)

		for _, li := range legal.Licenses() {
			fmt.Println(li)
			fmt.Println()
		}

		os.Exit(0)
	}
}
