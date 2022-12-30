// Package appsluggr packages a given binary into a Heroku-compatible slug.
package appsluggr

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"
	"github.com/rogpeppe/go-internal/dirhash"
)

// Must calls Pack and panics on an error.
//
// This is useful for using appsluggr from yeet scripts.
func Must(fname, outFname string) {
	if err := Pack(fname, outFname); err != nil {
		log.Panicf("error packing %s into %s: %v", fname, outFname, err)
	}
}

// Pack takes a statically linked binary and packs it into a Heroku-compatible slug image
// at outFname.
func Pack(fname string, outFname string) error {
	dir, err := os.MkdirTemp("", "appsluggr")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	fout, err := os.Create(outFname)
	if err != nil {
		return err
	}
	defer fout.Close()

	gzw := gzip.NewWriter(fout)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	os.MkdirAll(filepath.Join(dir, "bin"), 0777)

	if err := copy.Copy(fname, filepath.Join(dir, "bin", "main")); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "Procfile"), []byte("web: /app/bin/main\n"), 0777); err != nil {
		return err
	}

	hash, err := dirhash.HashDir(dir, os.Args[0], dirhash.Hash1)
	if err != nil {
		return err
	}

	gzw.Comment = hash
	log.Printf("hash: %s", hash)

	return filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		// return on any error
		if err != nil {
			log.Printf("got error on %s: %v", file, err)
			return err
		}

		if fi.IsDir() {
			return nil // not a file.  ignore.
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, dir, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !fi.Mode().IsRegular() {
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
}
