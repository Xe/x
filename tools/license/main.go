package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"text/template"
	"time"

	"github.com/Xe/x/tools/license/licenses"
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
}

func main() {
	flag.Parse()

	if *showAll {
		fmt.Println("Licenses available:")
		for license, _ := range licenses.List {
			fmt.Printf("  %s\n", license)
		}

		os.Exit(1)
	}

	if len(flag.Args()) != 1 {
		flag.Usage()
	}

	kind := flag.Arg(0)

	outfile := "LICENSE"

	var licensetext string
	if _, ok := licenses.List[kind]; !ok {
		fmt.Printf("invalid license kind %s\n", kind)
		os.Exit(1)
	}

	licensetext = licenses.List[kind]

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

	tmpl, err := template.New("license").Parse(licensetext)
	if err != nil {
		log.Fatal(err)
	}

	err = tmpl.Execute(wr, struct {
		Name  string
		Email string
		Year  int
	}{
		Name:  *name,
		Email: *email,
		Year:  time.Now().Year(),
	})
}
