package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	format    = flag.String("format", "jpg", "Image format to prefer")
	outputDir = flag.String("output", "./output", "Where to write validated images to")
	where     = flag.String("where", ".", "Directory to scan for unvalidated images")
	minWidth  = flag.Int("minsize", 2559, "Minimum width") // Width of SP3 display
	debugFlag = flag.Bool("debug", false, "panic() on error?")
)

func main() {
	flag.Parse()
	log.Printf("Debug: %v", *debugFlag)

	// discard error value. XXX fix this?
	os.Mkdir(*outputDir, 0755)

	err := filepath.Walk(*where, validate)
	if err != nil {
		if *debugFlag {
			panic(err)
		} else {
			log.Fatal(err)
		}
	}
}

// validate takes a walked directory entry and sees if the image is big enough.
func validate(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	fin, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fin.Close()

	if *debugFlag {
		log.Println(path)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("%q: %v", path, r)
		}
	}()

	img, format, err := image.DecodeConfig(fin)
	if err != nil {
		return err
	}

	if img.Width > *minWidth {
		err = CopyFile(path, *outputDir+"/"+filepath.Base(strings.TrimSuffix(info.Name(), filepath.Ext(path))+"."+format))
		if err != nil {
			return err
		}
	}

	return nil
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}

	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}

	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}

		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}

	err = copyFileContents(src, dst)
	return

}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}

	defer func() {
		cerr := out.Close()

		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return
	}

	err = out.Sync()
	return
}
