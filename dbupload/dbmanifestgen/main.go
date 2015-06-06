package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	serverloc = flag.String("serverloc", "http://static.xeserv.us/", "server to prepend to url paths")
	sourceloc = flag.String("sourceloc", "", "source URL for metadata generation")
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Printf("%s: <dir>\n", os.Args[0])
		flag.Usage()
	}

	if *sourceloc == "" {
		log.Fatal("Need a source location")
	}

	images, err := ioutil.ReadDir(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		if strings.HasSuffix(image.Name(), ".json") {
			log.Printf("Skipped %s...", image)
			continue
		}

		fout, err := os.Create(image.Name() + ".json")

		var i UploadImage
		i.Image.SourceURL = *sourceloc
		i.Image.ImageURL = *serverloc + filepath.Base(image.Name())

		outdata, err := json.MarshalIndent(&i, "", "\t")
		if err != nil {
			panic(err)
		}

		fout.Write(outdata)
		fout.Close()
	}
}
