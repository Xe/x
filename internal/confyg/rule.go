// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package confyg

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func Parse(file string, data []byte, r Reader, al Allower) (*FileSyntax, error) {
	fs, err := parse(file, data)
	if err != nil {
		return nil, err
	}

	var errs bytes.Buffer
	for _, x := range fs.Stmt {
		switch x := x.(type) {
		case *Line:
			ok := al.Allow(x.Token[0], false)
			if ok {
				r.Read(&errs, fs, x, x.Token[0], x.Token[1:])
				continue
			}

			fmt.Fprintf(&errs, "%s:%d: can't allow line verb %s", file, x.Start.Line, x.Token[0])

		case *LineBlock:
			if len(x.Token) > 1 {
				fmt.Fprintf(&errs, "%s:%d: unknown block type: %s\n", file, x.Start.Line, strings.Join(x.Token, " "))
				continue
			}
			ok := al.Allow(x.Token[0], true)
			if ok {
				for _, l := range x.Line {
					r.Read(&errs, fs, l, x.Token[0], l.Token)
				}
				continue
			}

			fmt.Fprintf(&errs, "%s:%d: can't allow line block verb %s", file, x.Start.Line, x.Token[0])
		}
	}

	if errs.Len() > 0 {
		return nil, errors.New(strings.TrimRight(errs.String(), "\n"))
	}
	return fs, nil
}

func isDirectoryPath(ns string) bool {
	// Because go.mod files can move from one system to another,
	// we check all known path syntaxes, both Unix and Windows.
	return strings.HasPrefix(ns, "./") || strings.HasPrefix(ns, "../") || strings.HasPrefix(ns, "/") ||
		strings.HasPrefix(ns, `.\`) || strings.HasPrefix(ns, `..\`) || strings.HasPrefix(ns, `\`) ||
		len(ns) >= 2 && ('A' <= ns[0] && ns[0] <= 'Z' || 'a' <= ns[0] && ns[0] <= 'z') && ns[1] == ':'
}

func isString(s string) bool {
	return s != "" && s[0] == '"'
}

func parseString(s *string) (string, error) {
	t, err := strconv.Unquote(*s)
	if err != nil {
		return "", err
	}
	*s = strconv.Quote(t)
	return t, nil
}
