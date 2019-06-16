//+build dev

package main

import "net/http"

var assets http.FileSystem = http.Dir("./static")
