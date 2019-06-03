package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var _static_gruvbox_css = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x91\xdf\x6e\xa4\x20\x14\xc6\xef\x79\x8a\x93\xcc\xcd\x6e\xa2\xcd\xa0\x75\x6a\xf1\x66\x5f\x85\xbf\x42\x2a\xe2\x22\xd3\x9d\x66\xe3\xbb\x6f\x44\xed\x28\xdb\x29\x5e\x48\x80\xf3\x7d\xe7\x7c\x3f\x4b\x4d\x0f\x7f\x11\x80\x72\x7d\xc8\x15\xb5\xa6\xfb\x20\x60\x5d\xef\xc6\x81\x72\x99\xdd\xb7\x0d\x02\xb0\xf4\x96\xff\x31\x22\x68\x02\x65\xed\xa5\x9d\xcf\x06\x2a\x84\xe9\x5b\x02\xc5\x7a\x60\xa9\x6f\x4d\x4f\x80\x5e\x83\x6b\xd0\x84\x10\x73\xe2\x23\x7a\x30\xca\xdf\x5a\xef\xae\xbd\x20\x70\x2a\xea\xf9\x9b\x0b\xb8\xeb\x9c\x27\x10\xd7\x49\x32\xc1\x58\x11\xeb\x06\x2f\x93\xb2\x7c\x7d\x7a\x2a\x79\x59\x97\x97\x83\x3d\x5e\xdc\x99\xf3\x42\x7a\x02\xe7\x28\x41\x33\xa0\x84\xf2\x60\xde\xe5\xbc\x7b\x37\xa3\x09\x52\x44\xd5\x4d\x8a\xe1\x4b\x51\x47\xa9\x2f\x7c\xb0\x28\xce\x05\x8e\x52\x1a\x67\xa0\x8b\x0c\x74\x99\x81\x7e\xce\x40\x57\x51\x66\x99\x36\x67\x2e\x04\x67\x09\x3c\xe1\x98\xc2\x3c\x75\xe7\xf8\xdb\xef\xab\x0b\xeb\x10\xb1\xaf\xbc\x93\x2a\x10\xc0\xc3\x0d\x46\xd7\x19\x01\x27\x26\xa8\x7c\x2d\xf7\xb9\x9d\x9f\x2a\x69\x01\x9f\x87\xdb\x61\xbe\xfd\xf1\x84\xd0\x2f\x2b\x85\xa1\xf0\x63\xf0\x52\x49\x3f\x2e\x1d\xe7\x23\xd7\xd2\x4a\x02\x9d\x69\x75\xf8\x19\x8d\x67\xeb\x35\xff\x65\x1d\x28\x28\xa6\x30\x7f\x69\x3e\x2f\x0f\x2c\xee\x29\x03\x4c\x28\xfe\x36\x26\xa9\xd4\x67\x62\x1b\xbf\xed\x49\xca\x67\x2d\xbc\x53\xda\x89\x7f\x43\xeb\xd0\xdd\x8e\xd9\xc3\x3e\xd4\xab\xaa\xc4\xcb\x41\xfe\x01\xc1\x65\x7d\xcd\x71\x57\x9d\xf0\xdc\xcf\xf1\x3f\xd5\x4b\x55\xf1\xea\xb9\x49\xd4\x53\xb6\x49\x42\xe9\xe5\x84\x26\xf4\x2f\x00\x00\xff\xff\x38\x40\x6e\x02\xa0\x03\x00\x00")

func static_gruvbox_css() ([]byte, error) {
	return bindata_read(
		_static_gruvbox_css,
		"static/gruvbox.css",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"static/gruvbox.css": static_gruvbox_css,
}
// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func func() ([]byte, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"static": &_bintree_t{nil, map[string]*_bintree_t{
		"gruvbox.css": &_bintree_t{static_gruvbox_css, map[string]*_bintree_t{
		}},
	}},
}}
