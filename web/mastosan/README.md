# mastosan

Package mastosan takes Mastodon flavored HTML and emits Slack-flavored
markdown.

It works by using an embedded Rust program compiled to WebAssembly. At
the time of writing this adds an extra 0.2 seconds to compile and run the
WebAssembly module, but this can probably be fixed with aggressive sync.Once
caching should this prove a problem in the real world.

Building this Rust program outside of Nix flakes is UNSUPPORTED.

To build the Rust module from inside the Nix flake:

    cargo install wasm-snip
    cargo build --release
    wasm-opt -Oz -o ./testdata/mastosan-pre-snip.wasm
    wasm-snip --skip-producers-section --snip-rust-panicking-code -i ./testdata/mastosan-pre-snip.wasm ./testdata/mastosan.wasm
    rm ./testdata/mastosan-pre-snip.wasm

This adds about two megabytes to the resulting binary, including the AOT
WebAssembly runtime wazero: https://wazero.io/


