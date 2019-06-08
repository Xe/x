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

var _static_gruvbox_css = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x91\xcf\x6e\xa3\x30\x10\xc6\xef\x7e\x8a\x91\x72\xd9\x95\xa0\x8a\xa1\xa4\xc4\xb9\xec\xab\xf8\xcf\x10\x5b\xc5\x98\x35\x4e\x37\xd5\x8a\x77\x5f\x61\xa0\x01\xb7\x5d\x73\xc0\x1a\xcf\x7c\x33\xf3\xfd\x2c\x37\x1d\xfc\x25\x00\x8d\xeb\x42\xde\x70\x6b\xda\x77\x06\xd6\x75\x6e\xe8\xb9\xc4\xec\x71\xbd\x10\x00\xcb\xef\xf9\x1f\xa3\x82\x66\x50\xd6\x1e\xed\x14\xeb\xb9\x52\xa6\xbb\x32\x28\x96\x80\xe5\xfe\x6a\x3a\x06\xfc\x16\xdc\x85\x8c\x84\x30\x36\x60\x8b\x32\x18\x37\xb7\x12\x5c\xbe\x5e\xbd\xbb\x75\x8a\xc1\x41\x95\xf5\xe9\x2c\x62\x9e\x70\xea\xfd\x73\x42\x51\x4f\xdf\x24\x2c\x5d\xeb\x3c\x83\x78\x0e\x28\x94\x10\x45\xac\xeb\x3d\x26\x65\xf9\x92\x7a\x28\x65\x59\x97\xa7\xdd\x98\x74\x9e\x52\x38\xaf\xd0\x33\x38\x46\x09\x9e\x01\x67\x5c\x06\xf3\x86\xd3\xed\xcd\x0c\x26\xa0\x8a\xaa\xab\x94\xa0\xa7\xa2\x8e\x52\x5f\xf4\xa1\xaa\x38\x16\x34\x4a\x69\x9a\x81\x2e\x32\xd0\x65\x06\xfa\x39\x03\x5d\x45\x99\xd9\x95\x5c\xb8\x10\x9c\x65\xf0\x44\xa3\x5b\xd3\xd6\xad\x93\xaf\xbf\x6f\x2e\x2c\x4b\xc4\xb9\xf2\x16\x9b\xc0\x80\xf6\x77\x18\x5c\x6b\x14\x1c\x84\xe2\x78\x2e\xb7\xfe\x1e\x9f\x2a\xb4\x40\x8f\xfd\x7d\xb7\xdf\x36\x3c\x12\xf2\xcb\xa2\x32\x1c\x7e\xf4\x1e\x1b\xf4\xc3\x3c\x71\x3e\x48\x8d\x16\x19\xb4\xe6\xaa\xc3\xcf\xd8\x78\x6a\xbd\xf8\x3f\x9f\x1d\x85\x46\x34\x54\xbe\x5c\x3e\x1e\x77\x2c\x1e\x2e\x03\x8c\x24\xfe\x56\x26\xa9\xd4\x87\x63\x2b\xbf\x35\x25\xe5\xb3\x14\x3e\x28\x6d\xc4\xff\x43\x6b\x37\xdd\x86\xd9\xb7\x73\x34\xe7\xa6\x52\x2f\x3b\xf9\x6f\x08\xce\xe7\x6b\x8e\x9b\xea\x84\xe7\x76\x8f\xcf\x54\x4f\x55\x25\xab\xe7\x4b\xa2\x9e\xb2\x4d\x1c\x4a\x1f\x47\x32\x92\x7f\x01\x00\x00\xff\xff\xab\xef\x35\x5a\xc8\x03\x00\x00")

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
