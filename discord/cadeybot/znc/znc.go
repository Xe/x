package znc

import (
	"bufio"
	"errors"
	"strings"
	"time"
)

const (
	dateFormat = `2006-01-02`
	timeFormat = `[15:04:05]`

	status = `***`
)

// LogMessage is an individual log message scraped from ZNC logs.
type LogMessage struct {
	Sent   time.Time
	Sender string
	Body   string
}

// Reader reads IRC log lines from a given io.Reader.
type Reader struct {
	S *bufio.Scanner
}

var (
	// ErrFailedFilter is returned when the znc message fails the matching filter.
	ErrFailedFilter = errors.New("znc: failed filter function")
)

// ReadOldLine does what you'd expect. Expects old style ZNC logs.
func (r Reader) ReadOldLine() (*LogMessage, error) {
	if !r.S.Scan() {
		return nil, r.S.Err()
	}

	result := LogMessage{}

	line := r.S.Text()
	sp := strings.SplitN(line, " ", 3)
	timeDate := sp[0] + " " + sp[1]
	t, err := time.Parse(dateFormat+" "+timeFormat, timeDate)
	if err != nil {
		return nil, err
	}
	result.Sent = t
	rest := sp[2]

	switch rest[0] {
	case '*':
		result.Sender = status
		result.Body = rest
	case '<':
		split := strings.SplitN(rest, " ", 2)
		result.Sender = split[0][1 : len(split[0])-1]
		result.Body = split[1]
	}

	return &result, nil
}
