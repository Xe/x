# Reaching the Unix Philosophy's Logical Conclusion with WebAssembly

Hey there! This is the example code I wrote for my talk at [GopherCon
EU](https://gophercon.eu). This consists of a few folders with code:

- `cmd`: Executable commands for the demo.
- `cmd/aiyou`: The WebAssembly runner. It connects to `cmd/yuechu` and
  exposes network connections as a filesystem. It is intended to run
  `wasip1/echoclient.wasm`.
- `cmd/yuechu`: The echo server that takes lines of inputs from
  network connections and feeds them to WebAssembly modules then sends
  the output back to the client. It runs `wasip1/promptreply.wasm` and
  `wasip1/xesitemd.wasm`.
- `wasip1`: A folder full of small demo programs. Each is built with
  makefile commands.
- `wasip1/echoclient.wasm`: A small Rust program that tries to connect
  to the echo server, prompts for a line of input, prints what it got
  back, and then exits.
- `wasip1/promptreply.wasm`: A small Rust program that reads input
  from standard in and then writes it to standard out.
- `wasip1/xesitemd.wasm`: My [blog's](https://xeiaso.net) markdown to
  HTML parser. It reads xesite-flavored markdown over standard input
  and returns HTML over standard output.

In order to build and run the code in this folder, you must be using
Nix and be inside a `nix develop` shell. You can build most files in
`wasip1` by using `make` such as like this:

```
make echoreply.wasm promptreply.wasm
```

If you have any questions, please [email
me](https://xeiaso.net/contact) or open an issue on this repo.
