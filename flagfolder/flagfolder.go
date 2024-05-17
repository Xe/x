// Package flagfolder parses a folder on the disk as if each file in it had the contents of a command line flag.
//
// This is mainly intended to be used with environments like Kubernetes where you have your secrets mounted as a filesystem.
package flagfolder

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/stoewer/go-strcase"
)

// ParseSet parses secrets in a single folder into the given *flag.FlagSet.
//
// By default this will attempt to correct for several styles of naming for files:
//
// * kebab-case (the default)
// * SHOUTING-KEBAB-CASE
// * snake_case
// * SHOUTING_SNAKE_CASE
// * camelCase
// * HammerCase
func ParseSet(secretLocation string, set *flag.FlagSet) error {
	var (
		data []byte
		err  error
	)

	set.VisitAll(func(f *flag.Flag) {
		if err != nil {
			return
		}

		for _, fname := range []string{
			filepath.Join(secretLocation, f.Name),
			filepath.Join(secretLocation, strcase.UpperKebabCase(f.Name)),
			filepath.Join(secretLocation, strcase.LowerCamelCase(f.Name)),
			filepath.Join(secretLocation, strcase.UpperCamelCase(f.Name)),
			filepath.Join(secretLocation, strcase.SnakeCase(f.Name)),
			filepath.Join(secretLocation, strcase.UpperSnakeCase(f.Name)),
		} {
			var ferr error
			data, ferr = os.ReadFile(fname)
			if ferr != nil {
				slog.Debug("can't read", "fname", fname, "err", err)
				if os.IsNotExist(ferr) {
					continue
				}
				continue
			}

			if ferr := f.Value.Set(string(data)); ferr != nil {
				err = fmt.Errorf("flagfolder: failed to set flag %q in %s with value %q", f.Name, fname, string(data))
			}
		}
	})

	return err
}

// Parse parses all files in every folder under /run/secrets as if they were command-line flags.
//
// This is most useful when you are using environments like Kubernetes where the path of least resistance
// is to mount your secrets as a filesystem. Mount all your secrets into the pod and then let it figure
// itself out!
//
// To use this effectively, ensure that your Pods and Deployments mount secrets as volumes like this:
//
//	volumes:
//	  - name: secret-volume
//	    secret:
//	    secretName: shell
//	containers:
//	  - name: shell
//	    image: ubuntu:latest
//	    volumeMounts:
//	      - name: secret-volume
//	        readOnly: true
//	        mountPath: "/run/secrets/shell"
//
// By default this will attempt to correct for several styles of naming for files:
//
// * kebab-case (the default)
// * SHOUTING-KEBAB-CASE
// * snake_case
// * SHOUTING_SNAKE_CASE
// * camelCase
// * HammerCase
func Parse() {
	stats, err := os.ReadDir("/run/secrets")
	if err != nil {
		slog.Debug("can't read from /run/secrets", "err", err)
		return
	}

	for _, stat := range stats {
		if !stat.IsDir() {
			continue
		}

		loc := filepath.Join("/run/secrets", stat.Name())

		if err := ParseSet(loc, flag.CommandLine); err != nil {
			slog.Error("can't parse folder", "folder", loc, "err", err)
		}
	}
}
