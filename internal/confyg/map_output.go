package confyg

import (
	"bytes"
	"strings"
)

// MapConfig is a simple wrapper around a map.
type MapConfig map[string][]string

// Allow accepts everything.
func (mc MapConfig) Allow(verb string, block bool) bool {
	return true
}

func (mc MapConfig) Read(errs *bytes.Buffer, fs *FileSyntax, line *Line, verb string, args []string) {
	mc[verb] = append(mc[verb], strings.Join(args, " "))
}
