package gluassh

import (
	"bytes"
)

type Result struct {
	Out    *bytes.Buffer
	Err    *bytes.Buffer
	Status int
}

func NewResult(outbuf *bytes.Buffer, errbuf *bytes.Buffer, status int) *Result {
	return &Result{
		Out:    outbuf,
		Err:    errbuf,
		Status: status,
	}
}

func (r *Result) Successful() bool {
	if r.Status == 0 {
		return true
	} else {
		return false
	}
}

func (r *Result) Failed() bool {
	return !r.Successful()
}
