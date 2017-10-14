//
// This is an example command tool you can build with realpath
// You can provide any number of path target. This tool will
// output every path to stdout
//
package main

import (
	"flag"
	"fmt"
	"github.com/yookoala/realpath"
	"os"
)

var targets []string

func init() {

	flag.Parse()

	// read directory from remaining argument
	// or use current directory
	if flag.NArg() > 0 {
		targets = flag.Args()
	} else {
		// get current targets
		target, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse current path: %s\n", err.Error())
		}

		targets = append(targets, target)
	}

	// quit if no target can be determined
	if len(targets) == 0 {
		os.Exit(1)
	}

}

func main() {
	// print the realpath result for each targets
	for _, target := range targets {
		p, err := realpath.Realpath(target)
		if err != nil {
			// output error
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}

		// normal output
		fmt.Fprintln(os.Stdout, p)
	}
}
