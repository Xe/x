package awssig

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func CanonicalHeaderValue(r *http.Request, name string) string {
	switch name {
	case "host":
		return collapseSpaces(strings.TrimSpace(r.Host))
	case "content-length":
		if r.ContentLength >= 0 {
			return strconv.FormatInt(r.ContentLength, 10)
		}
	}
	vals := r.Header.Values(http.CanonicalHeaderKey(name))
	trimmed := make([]string, len(vals))
	for i, val := range vals {
		trimmed[i] = collapseSpaces(strings.TrimSpace(val))
	}
	return strings.Join(trimmed, ",")
}

// collapseSpaces folds runs of spaces down to a single space, matching the
// canonicalization AWS applies to header values outside quoted strings. Spaces
// inside a double-quoted token are preserved verbatim.
func collapseSpaces(s string) string {
	if !strings.Contains(s, "  ") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace, inQuote := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
			prevSpace = false
			b.WriteByte(c)
			continue
		}
		if c == ' ' && !inQuote {
			if !prevSpace {
				b.WriteByte(' ')
			}
			prevSpace = true
			continue
		}
		b.WriteByte(c)
		prevSpace = false
	}
	return b.String()
}

// CanonicalQuery sorts by parameter name, then by value for repeated names,
// matching the AWS SDKs. Sorting the joined "key=value" strings instead would
// disagree with them whenever one name is a prefix of another whose next byte
// sorts below '=' (e.g. "list" vs "list-type").
func CanonicalQuery(values url.Values, exclude string) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == exclude {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(values))
	for _, k := range keys {
		ek := AWSURIEncode(k, true)
		vs := append([]string(nil), values[k]...)
		sort.Strings(vs)
		for _, val := range vs {
			pairs = append(pairs, ek+"="+AWSURIEncode(val, true))
		}
	}
	return strings.Join(pairs, "&")
}

// AWSURIEncode applies RFC 3986 encoding with AWS's unreserved set. When
// encodeSlash is false, '/' is left intact (used for path segments).
func AWSURIEncode(s string, encodeSlash bool) string {
	const upperhex = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '.', c == '_', c == '~':
			b.WriteByte(c)
		case c == '/':
			if encodeSlash {
				b.WriteString("%2F")
			} else {
				b.WriteByte('/')
			}
		default:
			b.WriteByte('%')
			b.WriteByte(upperhex[c>>4])
			b.WriteByte(upperhex[c&0x0f])
		}
	}
	return b.String()
}

// BuildCanonicalRequest renders the canonical request over exactly the given
// (already sorted, lowercase) signed-header names. Identical for SigV4 and
// SigV4A.
func BuildCanonicalRequest(r *http.Request, sortedSignedHeaders []string, payloadHash string, disablePathEscaping bool) string {
	var ch strings.Builder
	for _, h := range sortedSignedHeaders {
		ch.WriteString(h)
		ch.WriteByte(':')
		ch.WriteString(CanonicalHeaderValue(r, h))
		ch.WriteByte('\n')
	}

	return strings.Join([]string{
		r.Method,
		CanonicalURI(r, disablePathEscaping),
		CanonicalQuery(r.URL.Query(), "X-Amz-Signature"),
		ch.String(),
		strings.Join(sortedSignedHeaders, ";"),
		payloadHash,
	}, "\n")
}

// CanonicalURI renders the canonical URI. When disablePathEscaping is true
// (S3 style) the on-the-wire encoded path is used directly; otherwise the
// already-encoded path is encoded a second time, as AWS mandates for every
// non-S3 service.
func CanonicalURI(r *http.Request, disablePathEscaping bool) string {
	path := r.URL.EscapedPath()
	if path == "" {
		return "/"
	}
	if disablePathEscaping {
		return path
	}
	return AWSURIEncode(path, false)
}
