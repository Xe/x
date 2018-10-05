package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// Puush constants
const (
	PuushBase      = "https://puush.me/api/"
	PuushAuthURL   = "https://puush.me/api/auth/"
	PuushUploadURL = "https://puush.me/api/up/"
)

func puushLogin(key string) (string, bool) {
	r, err := http.PostForm(PuushAuthURL, url.Values{"k": {key}})
	if err != nil {
		fmt.Println(err)
		return "", false
	}
	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	info := strings.Split(string(body), ",")
	if info[0] == "-1" {
		return "", false
	}

	session := info[1]
	return session, true
}

func puush(session, fname string, fin io.Reader) (*url.URL, error) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	kwriter, err := w.CreateFormField("k")
	if err != nil {
		return nil, err
	}

	io.WriteString(kwriter, session)

	file, _ := ioutil.ReadAll(fin)

	h := md5.New()
	h.Write(file)

	cwriter, err := w.CreateFormField("c")
	if err != nil {
		return nil, err
	}
	io.WriteString(cwriter, fmt.Sprintf("%x", h.Sum(nil)))

	zwriter, err := w.CreateFormField("z")
	if err != nil {
		return nil, err
	}
	io.WriteString(zwriter, "poop") // They must think their protocol is shit

	fwriter, err := w.CreateFormFile("f", fname)
	if err != nil {
		return nil, err
	}
	fwriter.Write(file)

	w.Close()

	req, err := http.NewRequest("POST", "http://puush.me/api/up", buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	info := strings.Split(string(body), ",")
	if info[0] == "0" {
		return url.Parse(info[1])
	}

	return nil, errors.New("upload failed")
}
