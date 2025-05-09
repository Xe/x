// Package licenses is the list of licenses that this software supports.
package licenses

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"strings"
	"text/template"
	"time"
)

var (
	//go:embed data/*.txt
	licenses embed.FS
)

func List() ([]string, error) {
	var result []string

	if err := fs.WalkDir(licenses, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if path == "data" {
			return nil
		}

		fname := strings.TrimSuffix(path, ".txt")
		fname = strings.TrimPrefix(fname, "data/")

		result = append(result, fname)
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func Has(license string) bool {
	fin, err := licenses.Open("data/" + license + ".txt")
	if err != nil {
		return false
	}
	defer fin.Close()

	return true
}

func Hydrate(license, name, email string, sink io.Writer) error {
	tmpl, err := template.ParseFS(licenses, "data/"+license+".txt")
	if err != nil {
		return err
	}

	return tmpl.Execute(sink, struct {
		Name  string
		Email string
		Year  int
	}{
		Name:  name,
		Email: email,
		Year:  time.Now().Year(),
	})
}
