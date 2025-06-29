package relaydv1

import (
	"net/http"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func RequestLogFromRequest(r *http.Request, ipAddress, requestID, ja4 string) *RequestLog {
	result := &RequestLog{
		RequestDate: timestamppb.Now(),
		Host:        r.Host,
		Method:      r.Method,
		Path:        r.URL.Path,
		Query:       map[string]string{},
		Headers:     map[string]string{},
		RemoteIp:    ipAddress,
		Ja4:         ja4,
		RequestId:   requestID,
	}

	for k, v := range r.URL.Query() {
		result.Query[k] = strings.Join(v, ",")
	}

	for k, v := range r.Header {
		switch {
		case k == "Cookie", k == "Authorization":
			continue
		}
		result.Headers[k] = strings.Join(v, ",")
	}

	return result
}
