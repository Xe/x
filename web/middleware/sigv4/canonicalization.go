package sigv4

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func canonicalHeaderValue(r *http.Request, name string) string {
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

func canonicalQuery(values url.Values, exclude string) string {
	pairs := make([]string, 0, len(values))
	for k, vs := range values {
		if k == exclude {
			continue
		}
		ek := awsURIEncode(k, true)
		for _, val := range vs {
			pairs = append(pairs, ek+"="+awsURIEncode(val, true))
		}
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "&")
}

// awsURIEncode applies RFC 3986 encoding with AWS's unreserved set. When
// encodeSlash is false, '/' is left intact (used for path segments).
func awsURIEncode(s string, encodeSlash bool) string {
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
