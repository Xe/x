package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/cors"
	"within.website/ln/ex"
)

var (
	maxBytes = flag.Int64("max-playground-bytes", 75, "how many bytes of data should users be allowed to post to the playground?")
)

func doHTTP() error {
	http.Handle("/", doTemplate(indexTemplate))
	http.Handle("/docs", doTemplate(docsTemplate))
	http.Handle("/faq", doTemplate(faqTemplate))
	http.Handle("/play", doTemplate(playgroundTemplate))
	http.HandleFunc("/api/playground", runPlayground)

	return http.ListenAndServe(":"+*port, ex.HTTPLog(cors.Default().Handler(http.DefaultServeMux)))
}

func httpError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(struct {
		Error string `json:"err"`
	}{
		Error: err.Error(),
	})
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
		httpError(w, err, http.StatusBadGateway)
		return
	}

	comp, err := compile(string(data))
	if err != nil {
		httpError(w, fmt.Errorf("compliation error: %v", err), http.StatusBadRequest)
		return
	}

	er, err := run(comp.Binary)
	if err != nil {
		httpError(w, fmt.Errorf("runtime error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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

      <p><big>A simple, fast, open-source, complete and safe language for developing modern software for the web</big></p>

      <hr />

      <h2>Example Program</h2>

      <code><pre>h</pre></code>

      <p>Outputs:</p>

      <code><pre>h</pre></code>

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

      <h2>Simple</h2>

      <p>h has a <a href="https://github.com/Xe/x/blob/master/h/h.peg">simple grammar</a> that gzips to 117 bytes. Creating a runtime environment for h is so trivial just about anyone can do it.</p>

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
        <li>h is the backbone of my startup.</li>
      </ul>

      <hr />

      <h2>Open-Source</h2>

      <p>The h compiler and default runtime are <a href="https://github.com/Xe/x/tree/master/cmd/h">open-source</a> free software sent out into the <a href="https://creativecommons.org/choose/zero/">Public Domain</a>. You can use h for any purpose at all with no limitations or restrictions.</p>

      <hr />

      <footer>
        <center><p><i>From <a href="https://christine.website">Within</a></i></p></center>
      </footer>
    </main>
  </body>
</html>`

const docsTemplate = `<html>
  <head>
    <title>The h Programming Language - Docs</title>
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

      <h1>Documentation</h1>

      <p><big id="comingsoon">Coming soon...</big></p>
      <script>
        Date.prototype.addDays = function(days) {
          var date = new Date(this.valueOf());
          date.setDate(date.getDate() + days);
          return date;
        }

        let date = new Date();
        date = date.addDays(1);
        document.getElementById("comingsoon").innerHTML = "Coming " + date.toDateString();
      </script>

      <hr />

      <footer>
        <center><p><i>From <a href="https://christine.website">Within</a></i></p></center>
      </footer>
    </main>
  </body>
</html>`

const faqTemplate = `<html>
  <head>
    <title>The h Programming Language - FAQ</title>
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

      <h1>Frequently Asked Questions</h1>

      <h2>What are the instructions of h?</h2>

      <p>h supports the following instructions:</p>
      <ul>
        <li><code>h</code></li>
        <li><code>'</code></li>
      </ul>

      <p>All valid h instructions must be separated by a space (<code>\0x20</code> or the spacebar on your computer). No other forms of whitespace are permitted. Any other characters will render your program <a href="http://jbovlaste.lojban.org/dict/gentoldra">gentoldra</a>.</p>

      <h2>How do I install and use h?</h2>

      <p>With any computer running <a href="https://golang.org">Go</a> 1.11 or higher:</p>

      <code><pre>go get -u -v within.website/x/cmd/h</pre></code>

      Usage is simple:

      <code><pre>Usage of h:
  -config string
        configuration file, if set (see flagconfyg(4))
  -koan
        if true, print the h koan and then exit
  -license
        show software licenses?
  -manpage
        generate a manpage template?
  -max-playground-bytes int
        how many bytes of data should users be allowed to
        post to the playground? (default 75)
  -o string
        if specified, write the webassembly binary created
        by -p here
  -o-wat string
        if specified, write the uncompiled webassembly
        created by -p here
  -p string
        h program to compile/run
  -port string
        HTTP port to listen on
  -v    if true, print the version of h and then exit</pre></code>

      <h2>What version is h?</h2>

      <p>Version 1.0, this will hopefully be the only release.</p>

      <h2>What is the h koan?</h2>

      <p>And Jesus said unto the theologians, "Who do you say that I am?"</p>

      <p>They replied: "You are the eschatological manifestation of the ground of our being, the kerygma of which we find the ultimate meaning in our interpersonal relationships."</p>

      <p>And Jesus said "...What?"</p>

      <p>Some time passed and one of them spoke "h".</p>

      <p>Jesus was enlightened.</p>

      <h2>Why?</h2>

      <p>That's a good question. The following blogposts may help you understand this more:</p>

      <ul>
        <li><a href="https://christine.website/blog/the-origin-of-h-2015-12-14">The Origin of h</a></li>
        <li><a href="https://christine.website/blog/formal-grammar-of-h-2019-05-19">A Formal Grammar of h</a></li>
      </ul>

      <h2>Who wrote h?</h2>

      <p><a href="https://christine.website">Within</a></p>

      <hr />

      <footer>
        <center><p><i>From <a href="https://christine.website">Within</a></i></p></center>
      </footer>
    </main>
  </body>
</html>`

const playgroundTemplate = `<html>
  <head>
    <title>The h Programming Language - Playground</title>
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

      <h1>Playground</h1>

      <p><small>Unfortunately, Javascript is required to use this page, sorry.</small></p>

      <h2>Program</h2>

      <input id="program" type="text" value="h" />

      <input onClick="runProgram()" type="button" value="Run">
      <p id="status"></p>

      <h3>Output</h3>

      <code><pre id="output"></pre></code>

      <h4>WebAssembly Text Format</h4>

      <code><pre id="wat_box"></pre></code>

      <p>Gas used: <span id="gas_used"></span></p>
      <p>Execution time (nanoseconds): <span id="exec_time"></span></p>

      <h4>AST</h4>

      <code><pre id="ast_box"></pre></code>

      <script>
      function runProgram() {
        const programData = document.getElementById("program").value;
        const output = document.getElementById("output");
        const watBox = document.getElementById("wat_box");
        const astBox = document.getElementById("ast_box");
        const gasUsed = document.getElementById("gas_used");
        const execTime = document.getElementById("exec_time");
        const status = document.getElementById("status");

        status.innerHTML = "submitting to the server...";

        postData("/api/playground", programData)
          .then(function(data) {
             if (data.err != null) {
               status.innerHTML = data.err;
               return;
             }

             status.innerHTML = "success";
             watBox.innerHTML = data.prog.wat;
             astBox.innerHTML = data.prog.ast;
             output.innerHTML = data.res.out;
             gasUsed.innerHTML = data.res.gas;
             execTime.innerHTML = data.res.exec_duration;
          })
          .catch(function(error) {
             console.log(error);
             status.innerHTML = error + ". Please try again later?";
          });
      }

      function postData(url = "", data = "h") {
        return fetch(url, {
          method: "POST",
          mode: "cors",
          cache: "no-cache",
          headers: {
            "Content-Type": "text/plain",
          },
          referrer: "no-referrer",
          body: data,
        }).then(response => response.json());
      }
      </script>

      <hr />

      <footer>
        <center><p><i>From <a href="https://christine.website">Within</a></i></p></center>
      </footer>
    </main>
  </body>
</html>`
