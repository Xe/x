package main

import (
	"strings"

	"github.com/eaburns/peggy/peg"
	"within.website/x/cmd/hlang/h"
	"within.website/x/cmd/hlang/nguh"
)

// CompiledProgram is a fully parsed and compiled h program.
type CompiledProgram struct {
	Source string `json:"src"`
	Binary []byte `json:"bin"`
	AST    string `json:"ast"`
}

func compile(source string) (*CompiledProgram, error) {
	tree, err := h.Parse(source)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	err = peg.PrettyWrite(&sb, tree)
	if err != nil {
		return nil, err
	}

	wasmBytes, err := nguh.Compile(tree)
	if err != nil {
		return nil, err
	}

	result := CompiledProgram{
		Source: source,
		AST:    sb.String(),
		Binary: wasmBytes,
	}

	return &result, nil
}
