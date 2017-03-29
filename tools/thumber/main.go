package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

var (
	dirToWalk = flag.String("walkdir", "./in", "directory to walk and generate thumbnails for")
)

func main() {
	flag.Parse()

	err := filepath.Walk(*dirToWalk, makeThumbnail)
	if err != nil {
		log.Fatal(err)
	}
}

func makeThumbnail(fname string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	if strings.HasSuffix(fname, ".thumb.png") {
		return nil
	}

	if strings.HasSuffix(fname, ".html") {
		return nil
	}

	_, err = os.Stat("thumbs/" + filepath.Base(fname) + ".thumb.png")
	if err == nil {
		log.Printf("skipping %s", fname)
		return nil
	}

	log.Printf("Starting to open %s", fname)

	img, err := imaging.Open(fname)
	if err != nil {
		return err
	}

	croppedImage := imaging.Thumbnail(img, 256, 256, imaging.Lanczos)
	err = imaging.Save(croppedImage, "thumbs/"+filepath.Base(fname)+".thumb.png")

	return err
}
