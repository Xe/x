package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"text/template"
	"time"
)

var (
	name    = flag.String("name", "", "name of the person licensing the software")
	email   = flag.String("email", "", "email of the person licensing the software")
	outfile = flag.String("out", "LICENSE", "name of the file to write the output to")

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
		for license, _ := range licenses {
			fmt.Printf("  %s\n", license)
		}

		os.Exit(1)
	}

	if len(flag.Args()) != 1 {
		flag.Usage()
	}

	kind := flag.Arg(0)

	var licensetext string
	if _, ok := licenses[kind]; !ok {
		fmt.Printf("invalid license kind %s\n", kind)
		os.Exit(1)
	}

	licensetext = licenses[kind]

	if kind == "unlicense" && *outfile == "LICENSE" {
		*outfile = "UNLICENSE"
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

	fout, err := os.Create(*outfile)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	tmpl, err := template.New("license").Parse(licensetext)
	if err != nil {
		log.Fatal(err)
	}

	err = tmpl.Execute(fout, struct {
		Name  string
		Email string
		Year  int
	}{
		Name:  *name,
		Email: *email,
		Year:  time.Now().Year(),
	})
}
