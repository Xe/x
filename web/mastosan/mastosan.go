// Package mastosan takes Mastodon flavored HTML and emits Slack-flavored
// markdown.
//
// It works by using an embedded Rust program compiled to WebAssembly. At
// the time of writing this adds an extra 0.2 seconds to compile and run the
// WebAssembly module, but this can probably be fixed with aggressive sync.Once
// caching should this prove a problem in the real world.
//
// Building this Rust program outside of Nix flakes is UNSUPPORTED.
//
// To build the Rust module from inside the Nix flake:
//
//     cargo install wasm-snip
//     cargo build --release
//     wasm-opt -Oz -o ./testdata/mastosan-pre-snip.wasm
//     wasm-snip --skip-producers-section --snip-rust-panicking-code -i ./testdata/mastosan-pre-snip.wasm ./testdata/mastosan.wasm
//     rm ./testdata/mastosan-pre-snip.wasm
//
// This adds about two megabytes to the resulting binary, including the AOT
// WebAssembly runtime wazero: https://wazero.io/
package mastosan

import (
	"bytes"
	"context"
	_ "embed"
	"log"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed testdata/mastosan.wasm
var mastosanWasm []byte

// HTML2Slackdown converts a string full of HTML text to slack-flavored markdown.
//
// Internally this works by taking that HTML and piping it to a small Rust program
// using lol_html to parse the HTML and rejigger it into slack-flavored markdown.
// This has an added latency of about 0.2 seconds per invocation, but this is as
// fast as I can make it for now.
func HTML2Slackdown(ctx context.Context, text string) (string, error) {
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	fout := &bytes.Buffer{}
	fin := bytes.NewBufferString(text)

	config := wazero.NewModuleConfig().WithStdout(fout).WithStdin(fin).WithArgs("mastosan")

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	code, err := r.CompileModule(ctx, mastosanWasm)
	if err != nil {
		log.Panicln(err)
	}

	if _, err = r.InstantiateModule(ctx, code, config); err != nil {
		return "", err
	}

	return strings.TrimSpace(fout.String()), nil
}