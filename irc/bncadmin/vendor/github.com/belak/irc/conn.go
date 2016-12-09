package irc

import (
	"bufio"
	"fmt"
	"io"
)

// Conn represents a simple IRC client. It embeds an irc.Reader and an
// irc.Writer.
type Conn struct {
	*Reader
	*Writer
}

// NewConn creates a new Conn
func NewConn(rw io.ReadWriter) *Conn {
	// Create the client
	c := &Conn{
		NewReader(rw),
		NewWriter(rw),
	}

	return c
}

// Writer is the outgoing side of a connection.
type Writer struct {
	writer io.Writer
}

// NewWriter creates an irc.Writer from an io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w}
}

// Write is a simple function which will write the given line to the
// underlying connection.
func (w *Writer) Write(line string) error {
	_, err := w.writer.Write([]byte(line + "\r\n"))
	return err
}

// Writef is a wrapper around the connection's Write method and
// fmt.Sprintf. Simply use it to send a message as you would normally
// use fmt.Printf.
func (w *Writer) Writef(format string, args ...interface{}) error {
	return w.Write(fmt.Sprintf(format, args...))
}

// WriteMessage writes the given message to the stream
func (w *Writer) WriteMessage(m *Message) error {
	return w.Write(m.String())
}

// Reader is the incoming side of a connection. The data will be
// buffered, so do not re-use the io.Reader used to create the
// irc.Reader.
type Reader struct {
	reader *bufio.Reader
}

// NewReader creates an irc.Reader from an io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		bufio.NewReader(r),
	}
}

// ReadMessage returns the next message from the stream or an error.
func (r *Reader) ReadMessage() (*Message, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// Parse the message from our line
	return ParseMessage(line)
}
