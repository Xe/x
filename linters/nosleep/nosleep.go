// Package nosleep contains a Go linter that prevents the use of time.Sleep in tests.
//
// Time is not a synchronization mechanism. It is not an appropriate thing to use in
// tests because undoubtedly the thing you are synchronizing on is going to depend on
// some unspoken assumption that can and will randomly change. Do something else.
//
// If god is dead and you need to do this anyways, add a comment that reads:
//
//	// nosleep bypass(yournick): put a reason why here
package nosleep

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "nosleep",
	Doc:  "Prevent the use of time.Sleep in test functions",
	Run:  run,
}

var bypassRegex = regexp.MustCompile(`^// nosleep bypass\((\w+)\): ([\w \.\!\,]+)`)

// nodeFuncName converts an ast.CallExpr into the human-readable name of
// the function and name.
func nodeFuncName(node *ast.CallExpr) string {
	se, ok := node.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ie, ok := se.X.(*ast.Ident)
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s.%s", ie.Name, se.Sel.Name)
}

// run finds all time.Sleep calls and raises a linter warning unless there
// is an appropriate magic comment.
func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ignore := findBypassComments(pass.Fset, file.Comments)

		ast.Inspect(file, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			pos := pass.Fset.Position(ce.Pos())
			fname := pos.Filename
			line := pos.Line

			for _, ignoreLine := range ignore {
				if ignoreLine == line {
					return true
				}
			}

			if strings.HasSuffix(fname, "_test.go") {
				funcName := nodeFuncName(ce)
				if funcName == "time.Sleep" {
					pass.Reportf(ce.Pos(), "use of time.Sleep in testing code")
				}
			}

			return true
		})
	}

	return nil, nil
}

// findBypassComments finds all the comments in the file that meet the bypassRegex
// and then adds it to a slice of lines to ignore.
func findBypassComments(fset *token.FileSet, cg []*ast.CommentGroup) []int {
	var result []int
	for _, g := range cg {
		for _, c := range g.List {
			text := bypassRegex.FindString(c.Text)
			if text == "" {
				continue
			}

			lnum := fset.Position(c.Pos()).Line
			result = append(result, lnum)
		}
	}

	return result
}
