// Package documentfunctions implements a linter that forces you to document functions.
//
// When you are dealing with code that requires multiple people to collaborate, your
// code is not going to be "immediately obvious based on the implementation". You need
// to document all functions that you make. This can help find untested invariants that
// can lead to customers having a bad time.
//
// To disable this behavior, document your code.
package documentfunctions

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "documentfunctions",
	Doc:  "Prevent the use of time.Sleep in test functions",
	Run:  run,
}

var ignoreFunctions = []string{
	"main", // main functions are probably okay
}

// run runs the analysis pass for documentfunctions. It performs a depth-first
// traversal of all AST nodes in a file, looks for function definitions, and
// then ensures all of them have comments.
func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		pos := pass.Fset.Position(file.Pos())
		if !strings.HasSuffix(".go", pos.Filename) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			fe, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			for _, ignoreName := range ignoreFunctions {
				if fe.Name.Name == ignoreName {
					return true
				}
			}

			if fe.Doc == nil {
				if fe.Name.Name == "init" {
					pass.Reportf(fe.Pos(), "init functions must have side effects documented")
					return true
				}
				pass.Reportf(fe.Pos(), "function %s must have documentation", fe.Name.Name)
			}

			return true
		})
	}
	return nil, nil
}
