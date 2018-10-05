package tun2

import (
	"bufio"
	"context"
	"expvar"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Xe/ln"
	"github.com/Xe/ln/opname"
	failure "github.com/dgryski/go-failure"
	"github.com/pkg/errors"
	"github.com/xtaci/smux"
)

// Connection is a single active client -> server connection and session
// containing many streams over TCP+TLS or KCP+TLS. Every stream beyond the
// control stream is assumed to be passed to the underlying backend server.
//
// All Connection methods assume this is locked externally.
type Connection struct {
	id            string
	conn          net.Conn
	session       *smux.Session
	controlStream *smux.Stream
	user          string
	domain        string
	cf            context.CancelFunc
	detector      *failure.Detector
	Auth          *Auth
	usable        bool

	sync.Mutex
	counter *expvar.Int
}

func (c *Connection) cancel() {
	c.cf()
	c.usable = false
}

// F logs key->value pairs as an ln.Fer
func (c *Connection) F() ln.F {
	return map[string]interface{}{
		"id":     c.id,
		"remote": c.conn.RemoteAddr(),
		"local":  c.conn.LocalAddr(),
		"kind":   c.conn.LocalAddr().Network(),
		"user":   c.user,
		"domain": c.domain,
	}
}

// Ping ends a "ping" to the client. If the client doesn't respond or the connection
// dies, then the connection needs to be cleaned up.
func (c *Connection) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = opname.With(ctx, "tun2.Connection.Ping")
	ctx = ln.WithF(ctx, ln.F{"timeout": time.Second})

	req, err := http.NewRequest("GET", "http://backend/health", nil)
	if err != nil {
		panic(err)
	}
	req = req.WithContext(ctx)

	_, err = c.RoundTrip(req)
	if err != nil {
		ln.Error(ctx, err, c, ln.Action("pinging the backend"))
		return err
	}

	c.detector.Ping(time.Now())

	return nil
}

// OpenStream creates a new stream (connection) to the backend server.
func (c *Connection) OpenStream(ctx context.Context) (net.Conn, error) {
	ctx = opname.With(ctx, "OpenStream")
	if !c.usable {
		return nil, ErrNoSuchBackend
	}
	ctx = ln.WithF(ctx, ln.F{"timeout": time.Second})

	err := c.conn.SetDeadline(time.Now().Add(time.Second))
	if err != nil {
		ln.Error(ctx, err, c)
		return nil, err
	}

	stream, err := c.session.OpenStream()
	if err != nil {
		ln.Error(ctx, err, c)
		return nil, err
	}

	return stream, c.conn.SetDeadline(time.Time{})
}

// Close destroys resouces specific to the connection.
func (c *Connection) Close() error {
	err := c.controlStream.Close()
	if err != nil {
		return err
	}

	err = c.session.Close()
	if err != nil {
		return err
	}

	err = c.conn.Close()
	if err != nil {
		return err
	}

	return nil
}

// Connection-specific errors
var (
	ErrCantOpenSessionStream = errors.New("tun2: connection can't open session stream")
	ErrCantWriteRequest      = errors.New("tun2: connection stream can't write request")
	ErrCantReadResponse      = errors.New("tun2: connection stream can't read response")
)

// RoundTrip forwards a HTTP request to the remote backend and then returns the
// response, if any.
func (c *Connection) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	ctx = opname.With(ctx, "tun2.Connection.RoundTrip")
	stream, err := c.OpenStream(ctx)
	if err != nil {
		return nil, errors.Wrap(err, ErrCantOpenSessionStream.Error())
	}

	go func() {
		<-req.Context().Done()
		stream.Close()
	}()

	err = req.Write(stream)
	if err != nil {
		return nil, errors.Wrap(err, ErrCantWriteRequest.Error())
	}

	buf := bufio.NewReader(stream)

	resp, err := http.ReadResponse(buf, req)
	if err != nil {
		return nil, errors.Wrap(err, ErrCantReadResponse.Error())
	}

	c.counter.Add(1)

	return resp, nil
}
