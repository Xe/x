package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/eaburns/peggy/peg"
	"within.website/x/h"
)

var (
	wat2wasmLoc     string
	wasmTemplateObj *template.Template
)

func init() {
	loc, err := exec.LookPath("wat2wasm")
	if err != nil {
		panic(err)
	}

	wat2wasmLoc = loc
	wasmTemplateObj = template.Must(template.New("h.wast").Parse(wasmTemplate))
}

// CompiledProgram is a fully parsed and compiled h program.
type CompiledProgram struct {
	Source          string `json:"src"`
	WebAssemblyText string `json:"wat"`
	Binary          []byte `json:"bin"`
	AST             string `json:"ast"`
}

func compile(source string) (*CompiledProgram, error) {
	tree, err := h.Parse(source)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	err = peg.PrettyWrite(&sb, tree)

	result := CompiledProgram{
		Source: source,
		AST:    sb.String(),
	}

	dir, err := ioutil.TempDir("", "h")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	fout, err := os.Create(filepath.Join(dir, "h.wast"))
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)

	err = wasmTemplateObj.Execute(buf, []byte(tree.Text))
	if err != nil {
		return nil, err
	}

	result.WebAssemblyText = buf.String()
	_, err = fout.WriteString(result.WebAssemblyText)
	if err != nil {
		return nil, err
	}

	fname := fout.Name()

	err = fout.Close()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, wat2wasmLoc, fname, "-o", filepath.Join(dir, "h.wasm"))
	cmd.Dir = dir

	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filepath.Join(dir, "h.wasm"))
	if err != nil {
		return nil, err
	}

	result.Binary = data

	return &result, nil
}
