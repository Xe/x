package confyg

import "bytes"

// Reader is called when individual lines of the configuration file are being read.
// This is where you should populate any relevant structures with information.
//
// If something goes wrong in the file parsing step, add data to the errs buffer
// describing what went wrong.
type Reader interface {
	Read(errs *bytes.Buffer, fs *FileSyntax, line *Line, verb string, args []string)
}

// ReaderFunc implements Reader for inline definitions.
type ReaderFunc func(errs *bytes.Buffer, fs *FileSyntax, line *Line, verb string, args []string)

func (r ReaderFunc) Read(errs *bytes.Buffer, fs *FileSyntax, line *Line, verb string, args []string) {
	r(errs, fs, line, verb, args)
}
