// Command license is a simple software license generator.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"

	"github.com/posener/complete"
	"within.website/x/cmd/license/licenses"
	"within.website/x/internal"
)

var (
	name  = flag.String("name", "", "name of the person licensing the software")
	email = flag.String("email", "", "email of the person licensing the software")
	out   = flag.Bool("out", false, "write to a file instead of stdout")

	showAll = flag.Bool("show", false, "show all licenses instead of generating one")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s [options] <license kind>\n\n", os.Args[0])
		flag.PrintDefaults()

		fmt.Fprintln(os.Stderr, "\nBy default the name and email are scraped from `git config`")
		os.Exit(2)
	}

	names, err := licenses.List()
	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(names)

	internal.HandleCompletion(complete.PredictSet(names...), nil)
}

func main() {
	internal.HandleStartup()

	if *showAll {
		fmt.Println("Licenses available:")
		names, err := licenses.List()
		if err != nil {
			log.Fatal(err)
		}

		for _, name := range names {
			fmt.Println("*", name)
		}

		os.Exit(1)
	}

	if len(flag.Args()) != 1 {
		flag.Usage()
	}

	kind := flag.Arg(0)

	outfile := "LICENSE"

	if !licenses.Has(kind) {
		fmt.Printf("invalid license kind %s\n", kind)
		os.Exit(1)
	}

	if kind == "unlicense" && *out {
		outfile = "UNLICENSE"
	}

	if kind == "sqlite" && *out {
		outfile = "BLESSING"
	}

	if *name == "" {
		cmd := exec.Command("git", "config", "user.name")

		out, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}

		myname := string(out)
		*name = myname[:len(myname)-1]
	}

	if *email == "" {
		cmd := exec.Command("git", "config", "user.email")

		out, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}

		myemail := string(out)
		*email = myemail[:len(myemail)-1]
	}

	var wr io.Writer

	if *out {
		fout, err := os.Create(outfile)
		if err != nil {
			log.Fatal(err)
		}
		defer fout.Close()

		wr = fout
	} else {
		wr = os.Stdout
		defer fmt.Println()
	}

	if err := licenses.Hydrate(kind, *name, *email, wr); err != nil {
		log.Fatal(err)
	}
}
