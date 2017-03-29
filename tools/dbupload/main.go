package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	apiKeyLocation = flag.String("apikeyloc", "/home/xena/.local/share/within/db.key", "Derpibooru API key location")
	cookieLocation = flag.String("cookieloc", "/home/xena/.db.cookie", "Location for magic cookie")
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Printf("%s: <image to upload>\n", os.Args[0])
		fmt.Printf("All files must have a manifest json file.\n")
		flag.Usage()
	}

	dbkey, err := ioutil.ReadFile(*apiKeyLocation)
	if err != nil {
		panic(err)
	}

	mydbkey := strings.Split(string(dbkey), "\n")[0]

	cookie, err := ioutil.ReadFile(*cookieLocation)
	if err != nil {
		panic(err)
	}

	image := flag.Arg(0)

	if strings.HasSuffix(image, ".json") {
		log.Printf("Skipped %s...", image)
		return
	}

	metafin, err := os.Open(image + ".json")
	if err != nil {
		log.Fatal("image " + image + " MUST have description manifest for derpibooru")
	}
	defer metafin.Close()

	metabytes, err := ioutil.ReadAll(metafin)
	if err != nil {
		log.Fatal(err)
	}

	var meta UploadImage
	err = json.Unmarshal(metabytes, &meta)
	if err != nil {
		log.Fatal(err)
	}

	imfin, err := os.Open(image)
	if err != nil {
		log.Fatal("cannot open " + image)
	}
	defer imfin.Close()

	if meta.Image.ImageURL == "" {
		panic("need file uploaded somewhere?")
	}

	outmetabytes, err := json.Marshal(&meta)

	req, err := http.NewRequest("POST", "https://derpibooru.org/images.json?key="+mydbkey, bytes.NewBuffer(outmetabytes))
	if err != nil {
		panic(err)
	}

	c := &http.Client{}

	req.Header = http.Header{
		"User-Agent":   {"Xena's crappy upload tool"},
		"Cookie":       {string(cookie)},
		"Content-Type": {"application/json"},
	}

	resp, err := c.Do(req)
	if err != nil {
		log.Printf("%#v", err)
		fmt.Printf("Request ID: %s\n", resp.Header.Get("X-Request-Id"))
		return
	}

	if resp.StatusCode != 201 {
		io.Copy(os.Stdout, resp.Body)
		return
	}

	respbytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	var i Image
	json.Unmarshal(respbytes, &i)

	fmt.Printf("Uploaded as https://derpibooru.org/%d\n", i.IDNumber)

	time.Sleep(20 * time.Second)
}
