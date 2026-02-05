package entrypoint

import (
	"cmp"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"within.website/x/cmd/sakurajima/internal/config"
)

var (
	requestsRejectedByLimits = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "techaro",
		Subsystem: "osiris",
		Name:      "requests_rejected_by_limits_total",
	}, []string{"domain", "reason"})
)

// limitReason is the reason for rejecting a request.
type limitReason string

const (
	limitReasonRequestBodyTooLarge limitReason = "request_body_too_large"
	limitReasonHeadersTooLarge     limitReason = "headers_too_large"
	limitReasonTooManyHeaders      limitReason = "too_many_headers"
)

// WithLimits wraps an http.Handler with request size limits middleware.
func WithLimits(domain string, limits config.Limits, h http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		maxBodySize := limits.MaxRequestBodyBytes()
		maxHeaderSize := limits.MaxHeaderSizeBytes()
		maxHeaderCount := limits.MaxHeaderCountValue()

		// Check header count
		if len(r.Header) > maxHeaderCount {
			log.LogAttrs(r.Context(), slog.LevelWarn, "request rejected: too many headers",
				slog.String("domain", domain),
				slog.Int("header_count", len(r.Header)),
				slog.Int("max_header_count", maxHeaderCount),
			)
			requestsRejectedByLimits.WithLabelValues(domain, string(limitReasonTooManyHeaders)).Inc()
			http.Error(w, "Too Many Headers", http.StatusRequestHeaderFieldsTooLarge)
			return
		}

		// Check header size (approximate)
		headerSize := headerSizeBytes(r)
		if headerSize > maxHeaderSize {
			log.LogAttrs(r.Context(), slog.LevelWarn, "request rejected: headers too large",
				slog.String("domain", domain),
				slog.Int64("header_size", headerSize),
				slog.Int64("max_header_size", maxHeaderSize),
			)
			requestsRejectedByLimits.WithLabelValues(domain, string(limitReasonHeadersTooLarge)).Inc()
			http.Error(w, "Request Header Fields Too Large", http.StatusRequestHeaderFieldsTooLarge)
			return
		}

		// Limit request body size
		// We use MaxBytesReader which will return an error if the body exceeds the limit
		// We need to wrap the body before passing to the handler
		if r.Body != nil && r.ContentLength > maxBodySize {
			log.LogAttrs(r.Context(), slog.LevelWarn, "request rejected: request body too large",
				slog.String("domain", domain),
				slog.Int64("content_length", r.ContentLength),
				slog.Int64("max_body_size", maxBodySize),
			)
			requestsRejectedByLimits.WithLabelValues(domain, string(limitReasonRequestBodyTooLarge)).Inc()
			http.Error(w, "Payload Too Large", http.StatusRequestEntityTooLarge)
			return
		}

		if r.Body != nil {
			// Limit the body size - this will cause Read to return an error if exceeded
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		}

		h.ServeHTTP(w, r)
	})
}

// headerSizeBytes approximates the size of the request headers in bytes.
// This is an approximation as it doesn't account for exact HTTP wire format.
func headerSizeBytes(r *http.Request) int64 {
	var size int64
	for key, values := range r.Header {
		// Add header name + ": " size
		size += int64(len(key) + 2)
		for _, v := range values {
			// Add value + "\r\n" size
			size += int64(len(v) + 2)
		}
	}
	// Account for request line (e.g., "GET /path HTTP/1.1\r\n")
	size += int64(len(r.Method) + 1 + len(r.URL.Path) + 1 + len(r.Proto) + 2)
	// Account for final "\r\n"
	size += 2
	return size
}

// GetDomainLimits returns the limits for a domain, or defaults if not configured.
func GetDomainLimits(d config.Domain) config.Limits {
	if d.Limits == nil {
		return config.DefaultLimits()
	}
	// Merge with defaults for any unset values
	def := config.DefaultLimits()
	return config.Limits{
		MaxRequestBody: cmp.Or(d.Limits.MaxRequestBody, def.MaxRequestBody),
		MaxHeaderSize:  cmp.Or(d.Limits.MaxHeaderSize, def.MaxHeaderSize),
		MaxHeaderCount: cmp.Or(d.Limits.MaxHeaderCount, def.MaxHeaderCount),
	}
}
