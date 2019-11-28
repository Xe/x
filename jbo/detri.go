package jbo

import "time"

const (
	detriFormat           = `de'i li 2006 pi'e 01 pi'e 02`
	detriTemciFormat      = `de'i li 2006 pi'e 01 pi'e 02 ti'u 15 pi'e 04 pi'e 05`
	iOS13DetriFormat      = `Y2006 M01 2, Mon`
	iOS13DetriTemciFormat = `Y2006 M01 2, Mon 15:04`
)

// Detri formats a datestamp for Lojban.
func Detri(t time.Time) string {
	return t.Format(detriFormat)
}

// DetriTemci formats a timestamp for Lojban.
func DetriTemci(t time.Time) string {
	return t.Format(detriTemciFormat)
}

// IOS13Detri formats a datestamp like iOS 13 does with the Lojban locale.
func IOS13Detri(t time.Time) string {
	return t.Format(iOS13DetriFormat)
}

// IOS13DetriTemci formats a date/timestamp like iOS 13 does with the Lojban locale.
func IOS13DetriTemci(t time.Time) string {
	return t.Format(iOS13DetriTemciFormat)
}
