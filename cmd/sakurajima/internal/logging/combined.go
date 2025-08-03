package logging

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// LogHTTPRequest writes HTTP request data to a writer in Apache/nginx combined log format with vhost
// Format: vhost remoteaddr - remoteuser [timestamp] "method uri protocol" status bytes "referer" "user-agent" duration
func LogHTTPRequest(w io.Writer, r *http.Request, status int, bytesWritten int64, duration time.Duration) {
	// Extract various fields from the request
	vhost := r.Host
	if vhost == "" {
		vhost = "-"
	}

	remoteAddr := r.RemoteAddr
	if remoteAddr == "" {
		remoteAddr = "-"
	} else {
		// Split out the port from the address, keeping only the IP
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			remoteAddr = host
		}
		// If SplitHostPort fails, we keep the original remoteAddr
	}

	// Extract remote user from basic auth if available
	remoteUser := "-"
	if username, _, ok := r.BasicAuth(); ok && username != "" {
		remoteUser = username
	}

	// Format timestamp in Apache log format
	timestamp := time.Now().Format("02/Jan/2006:15:04:05 -0700")

	// Extract method, URI, and protocol
	method := r.Method
	if method == "" {
		method = "GET"
	}

	uri := r.RequestURI
	if uri == "" {
		if r.URL != nil {
			uri = r.URL.Path
			if r.URL.RawQuery != "" {
				uri += "?" + r.URL.RawQuery
			}
		}
		if uri == "" {
			uri = "/"
		}
	}

	protocol := r.Proto
	if protocol == "" {
		protocol = "HTTP/1.0"
	}

	// Extract headers
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "-"
	}

	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "-"
	}

	// Format duration in milliseconds (more standard for web server logs)
	durationMillis := duration.Milliseconds()

	// Write the log entry in combined log format with vhost and duration
	fmt.Fprintf(w, "%s %s - %s [%s] \"%s %s %s\" %d %d \"%s\" \"%s\" %d\n",
		vhost, remoteAddr, remoteUser, timestamp, method, uri, protocol,
		status, bytesWritten, referer, userAgent, durationMillis)
}
