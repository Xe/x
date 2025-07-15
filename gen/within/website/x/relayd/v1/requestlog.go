package relaydv1

import (
	"net/http"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func RequestLogFromRequest(r *http.Request, ipAddress, requestID string, fingerprints map[string]string) *RequestLog {
	result := &RequestLog{
		RequestDate:  timestamppb.Now(),
		Host:         r.Host,
		Method:       r.Method,
		Path:         r.URL.Path,
		Query:        map[string]string{},
		Headers:      map[string]string{},
		RemoteIp:     ipAddress,
		Ja3N:         fingerprints["ja3n"],
		Ja4:          fingerprints["ja4"],
		RequestId:    requestID,
		Fingerprints: fingerprints,
	}

	for k, v := range r.URL.Query() {
		result.Query[k] = strings.Join(v, ",")
	}

	for k, v := range r.Header {
		switch {
		case k == "Cookie", k == "Authorization":
			continue
		case strings.HasPrefix(k, "X-"):
			continue
		}
		result.Headers[k] = strings.Join(v, ",")
	}

	return result
}
