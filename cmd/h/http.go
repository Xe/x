package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	maxBytes = flag.Int64("max-playground-bytes", 75, "how many bytes of data should users be allowed to post to the playground?")
)

func doHTTP() error {
	http.Handle("/", doTemplate(indexTemplate))
	http.HandleFunc("/api/playground", runPlayground)

	return http.ListenAndServe(":"+*port, nil)
}

func runPlayground(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	rc := http.MaxBytesReader(w, r.Body, *maxBytes)
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		http.Error(w, "too many bytes sent", http.StatusBadRequest)
		return
	}

	comp, err := compile(string(data))
	if err != nil {
		http.Error(w, fmt.Sprintf("compilation error: %v", err), http.StatusBadRequest)
		return
	}

	er, err := run(comp.Binary)
	if err != nil {
		http.Error(w, fmt.Sprintf("runtime error: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(struct {
		Program *CompiledProgram `json:"prog"`
		Results *ExecResult      `json:"res"`
	}{
		Program: comp,
		Results: er,
	})
}

func doTemplate(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, body)
	})
}

const indexTemplate = `<html>
  <head>
    <title>The h Programming Language</title>
    <link rel="stylesheet" href="https://within.website/static/gruvbox.css">
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  </head>
  <body>
    <main>
      <nav>
        <a href="/">The h Programming Language</a> -
        <a href="/docs">Docs</a> -
        <a href="/play">Playground</a> -
        <a href="/faq">FAQ</a>
      </nav>

      <h1>The h Programming Language</h1>

      <p>A simple, fast, complete and safe language for developing modern software for the web</p>

      <hr />

      <h2>Example Program</h2>

      <code>
      h
      </code>

      <hr />

      <h2>Fast Compilation</h2>

      <p>h compiles hundreds of characters of source per second. I didn't really test how fast it is, but when I was testing it the speed was fast enough that I didn't care to profile it.</p>

      <hr />

      <h2>Safety</h2>

      <p>h is completely memory safe with no garbage collector or heap allocations. It does not allow memory leaks to happen, nor do any programs in h have the possibility to allocate memory.</p>

      <ul>
        <li>No null</li>
        <li>Completely deterministic behavior</li>
        <li>No mutable state</li>
        <li>No persistence</li>
        <li>All functions are pure functions</li>
        <li>No sandboxing required</li>
      </ul>

      <hr />

      <h2>Zero* Dependencies</h2>

      <p>h generates <a href="http://webassembly.org">WebAssembly</a>, so every binary produced by the compiler is completely dependency free save a single system call: <code>h.h</code>. This allows for modern, future-proof code that will work on all platforms.</p>

      <hr />

      <h2>Platform Support</h2>

      <p>h supports the following platforms:</p>

      <ul>
        <li>Google Chrome</li>
        <li>Electron</li>
        <li>Chromium Embedded Framework</li>
        <li>Microsoft Edge</li>
        <li>Olin</li>
      </ul>

      <hr />

      <h2>International Out of the Box</h2>

      <p>h supports multiple written and spoken languages with true contextual awareness. It not only supports the Latin <code>h</code> as input, it also accepts the <a href="http://lojban.org">Lojbanic</a> <code>'</code> as well. This allows for full 100% internationalization into Lojban should your project needs require it.</p>

      <hr />

      <h2>Testimonials</h2>

      <p>Not convinced? Take the word of people we probably didn't pay for their opinion.</p>

      <ul>
        <li>I don't see the point of this.</li>
        <li>This solves all my problems. All of them. Just not in the way I expected it to.</li>
        <li>Yes.</li>
        <li>Perfect.</li>
      </ul>
    </main>
  </body>
</html>`
