package main

import (
	"errors"
	"net"
	"sync"
	"testing"
)

// scriptedListener returns a transient Accept error, then conn, then
// net.ErrClosed.
type scriptedListener struct {
	mu      sync.Mutex
	conn    net.Conn
	accepts int
}

func (l *scriptedListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.accepts++
	switch l.accepts {
	case 1:
		return nil, errors.New("accept: too many open files")
	case 2:
		conn := l.conn
		l.conn = nil
		return conn, nil
	default:
		return nil, net.ErrClosed
	}
}

func (l *scriptedListener) Close() error   { return nil }
func (l *scriptedListener) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4zero} }

func TestHandleRetriesAcceptErrors(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()

	l := &scriptedListener{conn: server}
	s := &Server{}

	errCh := make(chan error, 1)
	// "invalid" makes net.Dial fail immediately during address parsing, so the
	// handler goroutine cannot linger on a real connection attempt.
	go func() { errCh <- s.Handle(l, "invalid", "http") }()

	err := <-errCh
	if !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Handle returned %v, want net.ErrClosed", err)
	}

	l.mu.Lock()
	accepts := l.accepts
	l.mu.Unlock()
	if accepts != 3 {
		t.Fatalf("Accept called %d times, want 3", accepts)
	}
}

func TestCloseWriteNonTCPConn(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	closeWrite(a)
}
