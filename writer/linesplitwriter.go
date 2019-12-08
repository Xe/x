package writer

import (
	"bytes"
	"io"
)

func LineSplitting(sink io.Writer) io.Writer {
	return lineSplittingWriter{writer: sink}
}

type lineSplittingWriter struct {
	writer io.Writer
}

func (lsw lineSplittingWriter) Write(p []byte) (int, error) {
	for _, line := range bytes.Split(p, []byte("\n")) {
		if len(line) == 0 {
			continue
		}

		_, err := lsw.writer.Write(append(line, '\n'))
		if err != nil {
			return -1, err
		}
	}

	return len(p), nil
}
