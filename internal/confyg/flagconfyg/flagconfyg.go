// Package flagconfyg is a hack around confyg. This will blindly convert config
// verbs to flag values.
package flagconfyg

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"log"
	"strings"

	"within.website/x/internal/confyg"
)

// CmdParse is a quick wrapper for command usage. It explodes on errors.
func CmdParse(ctx context.Context, path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = Parse(path, data, flag.CommandLine)
	if err != nil {
		log.Printf("can't parse %s: %v", path, err)
		return
	}
}

// Parse parses the config file in the given file by name, bytes data and into
// the given flagset.
func Parse(name string, data []byte, fs *flag.FlagSet) error {
	lineRead := func(errs *bytes.Buffer, fs_ *confyg.FileSyntax, line *confyg.Line, verb string, args []string) {
		err := fs.Set(verb, strings.Join(args, " "))
		if err != nil {
			errs.WriteString(err.Error())
		}
	}

	_, err := confyg.Parse(name, data, confyg.ReaderFunc(lineRead), confyg.AllowerFunc(allower))
	return err
}

func allower(verb string, block bool) bool {
	return true
}
