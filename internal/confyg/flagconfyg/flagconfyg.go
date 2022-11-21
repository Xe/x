// Package flagconfyg is a hack around confyg. This will blindly convert config
// verbs to flag values.
package flagconfyg

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"strings"

	"within.website/ln"
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
		ln.Error(ctx, err)
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

// Dump turns a flagset's values into a configuration file.
func Dump(fs *flag.FlagSet) []byte {
	result := &confyg.FileSyntax{
		Name: fs.Name(),
		Comments: confyg.Comments{
			Before: []confyg.Comment{
				{
					Token: "// generated from " + fs.Name() + " flags",
				}, {},
			},
		},
		Stmt: []confyg.Expr{},
	}

	fs.Visit(func(fl *flag.Flag) {
		commentTokens := []string{"//", fl.Usage}

		l := &confyg.Line{
			Comments: confyg.Comments{
				Suffix: []confyg.Comment{
					{
						Token: strings.Join(commentTokens, " "),
					},
				},
			},
			Token: []string{fl.Name, fl.Value.String()},
		}

		result.Stmt = append(result.Stmt, l)
	})

	return confyg.Format(result)
}
