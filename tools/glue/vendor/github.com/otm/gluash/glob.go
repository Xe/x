package gluash

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Glob files and directories
func Glob(pattern string) (matches []string, err error) {
	if !hasMeta(pattern) {
		if _, err = os.Lstat(pattern); err != nil {
			return nil, nil
		}
		return []string{pattern}, nil
	}
	dir, file := filepath.Split(pattern)
	switch dir {
	case "":
		dir = "."
	case string(filepath.Separator):
		// nothing
	default:
		dir = dir[0 : len(dir)-1] // chop off trailing separator
	}
	if !hasMeta(dir) {
		return glober(dir, file, nil)
	}
	var m []string
	m, err = Glob(dir)
	if err != nil {
		return
	}
	for _, d := range m {
		matches, err = glober(d, file, matches)
		if err != nil {
			return
		}
	}
	return
}

// glob searches for files matching pattern in the directory dir
// and appends them to matches. If the directory cannot be
// opened, it returns the existing matches. New matches are
// added in lexicographical order.
func glober(dir, pattern string, matches []string) (m []string, e error) {
	m = matches
	// fi, err := os.Stat(dir)
	// if err != nil {
	// 	return
	// }
	// if !fi.IsDir() {
	// 	return
	// }
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	defer d.Close()
	names, _ := d.Readdirnames(-1)
	sort.Strings(names)
	for _, n := range names {
		matched, err := filepath.Match(pattern, n)
		if err != nil {
			return m, err
		}
		if matched {
			m = append(m, filepath.Join(dir, n))
		}
	}
	return
}

// hasMeta reports whether path contains any of the magic characters
// recognized by Match.
func hasMeta(path string) bool {
	// TODO(niemeyer): Should other magic characters be added here?
	return strings.IndexAny(path, "*?[") >= 0
}
