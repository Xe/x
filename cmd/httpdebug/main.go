package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"within.website/x/internal"
)

var (
	bind          = flag.String("bind", ":3000", "TCP port to bind to")
	silent        = flag.Bool("silent", false, "if set, don't log http headers")
	maxHeaderSize = 64 * 1024 // 64KB limit for headers
)

func main() {
	internal.HandleStartup()

	slog.Info("listening", "url", "http://localhost"+*bind)
	log.Fatal(http.ListenAndServe(*bind, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.within/health" {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintln(w, "OK")
			return
		}

		// Validate and sanitize RequestURI
		sanitizedURI := r.RequestURI
		if !utf8.ValidString(sanitizedURI) {
			sanitizedURI = "[invalid UTF-8]"
		}
		if len(sanitizedURI) > 1024 {
			sanitizedURI = sanitizedURI[:1024] + "...[truncated]"
		}

		contains := strings.Contains(r.Header.Get("Accept"), "text/html")
		slog.Info("got request", "method", r.Method, "path", sanitizedURI)

		// Set Content-Type based on response format
		if contains {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, "<pre id=\"main\"><code>")
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}

		var out io.Writer

		switch *silent {
		case true:
			out = w
		case false:
			out = io.MultiWriter(w, os.Stdout)
		}

		// Write request line
		fmt.Fprintln(out, r.Method, sanitizedURI)

		// Write headers with basic size limit and error handling
		headerSize := 0
		for key, values := range r.Header {
			headerLine := key + ": " + strings.Join(values, ", ") + "\r\n"
			headerSize += len(headerLine)
			if headerSize > maxHeaderSize {
				fmt.Fprintln(out, "[... headers truncated due to size limit ...]")
				break
			}
			fmt.Fprint(out, headerLine)
		}

		// Add blank line to separate headers from body
		fmt.Fprintln(out)

		if contains {
			fmt.Fprintln(w, "</code></pre>")
		}
	})))
}
