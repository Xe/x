package internal

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/Xe/x/tools/license/licenses"
	"go4.org/legal"
)

// current working directory and date:time tag of app boot (useful for tagging slugs)
var (
	WD      string
	DateTag string
)

func init() {
	lwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	WD = lwd
	DateTag = time.Now().Format("010220061504")
}

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

// ShouldWork explodes if the given command with the given env, working dir and context fails.
func ShouldWork(ctx context.Context, env []string, dir string, cmdName string, args ...string) {
	loc, err := exec.LookPath(cmdName)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.CommandContext(ctx, loc, args...)
	cmd.Dir = dir
	cmd.Env = env

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("starting process, env: %v, pwd: %s, cmd: %s, args: %v", env, dir, loc, args)
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
