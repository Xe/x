package main

import "embed"

// content holds the render assets (GPL-2.0, see assets/README.md) and the
// static files (CSS, fonts) served to browsers.
//
//go:embed assets static
var content embed.FS
