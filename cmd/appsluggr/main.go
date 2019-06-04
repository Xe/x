package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"
	"within.website/x/internal"
)

var (
	web         = flag.String("web", "", "path to binary for web process")
	webScale    = flag.Int("web-scale", 1, "default scale for web process if defined")
	worker      = flag.String("worker", "", "path to binary for worker process")
	workerScale = flag.Int("worker-scale", 1, "default scale for worker process if defined")
	fname       = flag.String("fname", "slug.tar.gz", "slug name")
)

func main() {
	internal.HandleStartup()

	fout, err := os.Create(*fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	gzw := gzip.NewWriter(fout)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	dir, err := ioutil.TempDir("", "appsluggr")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up

	os.MkdirAll(filepath.Join(dir, "bin"), 0777)
	var procfile, scalefile string

	copy.Copy("translations", filepath.Join(dir, "translations"))
	if *web != "" {
		procfile += "web: /app/bin/web\n"
		scalefile += fmt.Sprintf("web=%d", *webScale)
		Copy(*web, filepath.Join(dir, "bin", "web"))

	}
	if *worker != "" {
		procfile += "worker: /app/bin/worker\n"
		scalefile += fmt.Sprintf("worker=%d", *workerScale)
		Copy(*worker, filepath.Join(dir, "bin", "worker"))
	}

	os.MkdirAll(filepath.Join(dir, ".config"), 0777)

	err = ioutil.WriteFile(filepath.Join(dir, ".buildpacks"), []byte("https://github.com/ryandotsmith/null-buildpack"), 0666)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(dir, "DOKKU_SCALE"), []byte(scalefile), 0666)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(dir, "Procfile"), []byte(procfile), 0666)
	if err != nil {
		log.Fatal(err)
	}

	filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
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

// Copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, st.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
