package tun2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Xe/ln"
	"github.com/Xe/ln/opname"
	failure "github.com/dgryski/go-failure"
	"github.com/pborman/uuid"
	cmap "github.com/streamrail/concurrent-map"
	"github.com/xtaci/smux"
)

// Error values
var (
	ErrNoSuchBackend             = errors.New("tun2: there is no such backend")
	ErrAuthMismatch              = errors.New("tun2: authenication doesn't match database records")
	ErrCantRemoveWhatDoesntExist = errors.New("tun2: this connection does not exist, cannot remove it")
)

// gen502Page creates the page that is shown when a backend is not connected to a given route.
func gen502Page(req *http.Request) *http.Response {
	template := `<html><head><title>no backends connected</title></head><body><h1>no backends connected</h1><p>Please ensure a backend is running for ${HOST}. This is request ID ${REQ_ID}.</p></body></html>`

	resbody := []byte(os.Expand(template, func(in string) string {
		switch in {
		case "HOST":
			return req.Host
		case "REQ_ID":
			return req.Header.Get("X-Request-Id")
		}

		return "<unknown>"
	}))
	reshdr := req.Header
	reshdr.Set("Content-Type", "text/html; charset=utf-8")

	resp := &http.Response{
		Status:     fmt.Sprintf("%d Bad Gateway", http.StatusBadGateway),
		StatusCode: http.StatusBadGateway,
		Body:       ioutil.NopCloser(bytes.NewBuffer(resbody)),

		Proto:         req.Proto,
		ProtoMajor:    req.ProtoMajor,
		ProtoMinor:    req.ProtoMinor,
		Header:        reshdr,
		ContentLength: int64(len(resbody)),
		Close:         true,
		Request:       req,
	}

	return resp
}

// ServerConfig ...
type ServerConfig struct {
	SmuxConf *smux.Config
	Storage  Storage
}

// Storage is the minimal subset of features that tun2's Server needs out of a
// persistence layer.
type Storage interface {
	HasToken(ctx context.Context, token string) (user string, scopes []string, err error)
	HasRoute(ctx context.Context, domain string) (user string, err error)
}

// Server routes frontend HTTP traffic to backend TCP traffic.
type Server struct {
	cfg    *ServerConfig
	ctx    context.Context
	cancel context.CancelFunc

	connlock sync.Mutex
	conns    map[net.Conn]*Connection

	domains cmap.ConcurrentMap
}

// NewServer creates a new Server instance with a given config, acquiring all
// relevant resources.
func NewServer(cfg *ServerConfig) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("tun2: config must be specified")
	}

	if cfg.SmuxConf == nil {
		cfg.SmuxConf = smux.DefaultConfig()

		cfg.SmuxConf.KeepAliveInterval = time.Second
		cfg.SmuxConf.KeepAliveTimeout = 15 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = opname.With(ctx, "tun2.Server")

	server := &Server{
		cfg: cfg,

		conns:   map[net.Conn]*Connection{},
		domains: cmap.New(),
		ctx:     ctx,
		cancel:  cancel,
	}

	go server.phiDetectionLoop(ctx)

	return server, nil
}

// Close stops the background tasks for this Server.
func (s *Server) Close() {
	s.cancel()
}

// Wait blocks until the server context is cancelled.
func (s *Server) Wait() {
	for {
		select {
		case <-s.ctx.Done():
			return
		}
	}
}

// Listen passes this Server a given net.Listener to accept backend connections.
func (s *Server) Listen(l net.Listener) {
	ctx := opname.With(s.ctx, "Listen")

	f := ln.F{
		"listener_addr":    l.Addr(),
		"listener_network": l.Addr().Network(),
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := l.Accept()
		if err != nil {
			ln.Error(ctx, err, f, ln.Action("accept connection"))
			continue
		}

		ln.Log(ctx, f, ln.Action("new backend client connected"), ln.F{
			"conn_addr":    conn.RemoteAddr(),
			"conn_network": conn.RemoteAddr().Network(),
		})

		go s.HandleConn(ctx, conn)
	}
}

// phiDetectionLoop is an infinite loop that will run the [phi accrual failure detector]
// for each of the backends connected to the Server. This is fairly experimental and
// may be removed.
//
// [phi accrual failure detector]: https://dspace.jaist.ac.jp/dspace/handle/10119/4784
func (s *Server) phiDetectionLoop(ctx context.Context) {
	ctx = opname.With(ctx, "phiDetectionLoop")
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := time.Now()

			s.connlock.Lock()
			for _, c := range s.conns {
				failureChance := c.detector.Phi(now)
				const thresh = 0.9 // the threshold for phi failure detection causing logs

				if failureChance > thresh {
					ln.Log(ctx, c, ln.Info("phi failure detection"), ln.F{
						"value":     failureChance,
						"threshold": thresh,
					})
				}
			}
			s.connlock.Unlock()
		}
	}
}

// backendAuthv1 runs a simple backend authentication check. It expects the
// client to write a json-encoded instance of Auth. This is then checked
// for token validity and domain matching.
//
// This returns the user that was authenticated and the domain they identified
// with.
func (s *Server) backendAuthv1(ctx context.Context, st io.Reader) (string, *Auth, error) {
	ctx = opname.With(ctx, "backendAuthv1")
	f := ln.F{
		"backend_auth_version": 1,
	}

	f["stage"] = "json decoding"

	d := json.NewDecoder(st)
	var auth Auth
	err := d.Decode(&auth)
	if err != nil {
		ln.Error(ctx, err, f)
		return "", nil, err
	}

	f["auth_domain"] = auth.Domain
	f["stage"] = "checking domain"

	routeUser, err := s.cfg.Storage.HasRoute(ctx, auth.Domain)
	if err != nil {
		ln.Error(ctx, err, f)
		return "", nil, err
	}

	f["route_user"] = routeUser
	f["stage"] = "checking token"

	tokenUser, scopes, err := s.cfg.Storage.HasToken(ctx, auth.Token)
	if err != nil {
		ln.Error(ctx, err, f)
		return "", nil, err
	}

	f["token_user"] = tokenUser
	f["stage"] = "checking token scopes"

	ok := false
	for _, sc := range scopes {
		if sc == "connect" {
			ok = true
			break
		}
	}

	if !ok {
		ln.Error(ctx, ErrAuthMismatch, f)
		return "", nil, ErrAuthMismatch
	}

	f["stage"] = "user verification"

	if routeUser != tokenUser {
		ln.Error(ctx, ErrAuthMismatch, f)
		return "", nil, ErrAuthMismatch
	}

	return routeUser, &auth, nil
}

// HandleConn starts up the needed mechanisms to relay HTTP traffic to/from
// the currently connected backend.
func (s *Server) HandleConn(ctx context.Context, c net.Conn) {
	ctx = opname.With(ctx, "HandleConn")
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	f := ln.F{
		"local":  c.LocalAddr().String(),
		"remote": c.RemoteAddr().String(),
	}

	session, err := smux.Server(c, s.cfg.SmuxConf)
	if err != nil {
		ln.Error(ctx, err, f, ln.Action("establish server side of smux"))

		return
	}
	defer session.Close()

	controlStream, err := session.OpenStream()
	if err != nil {
		ln.Error(ctx, err, f, ln.Action("opening control stream"))

		return
	}
	defer controlStream.Close()

	user, auth, err := s.backendAuthv1(ctx, controlStream)
	if err != nil {
		return
	}

	connection := &Connection{
		id:       uuid.New(),
		conn:     c,
		session:  session,
		user:     user,
		domain:   auth.Domain,
		cf:       cancel,
		detector: failure.New(15, 1),
		Auth:     auth,
	}
	connection.counter = expvar.NewInt("http.backend." + connection.id + ".hits")

	defer func() {
		if r := recover(); r != nil {
			ln.Log(ctx, connection, ln.F{"action": "connection handler panic", "err": r})
		}
	}()

	ln.Log(ctx, connection, ln.Action("backend successfully connected"))
	s.addConn(ctx, connection)

	connection.Lock()
	connection.usable = true // XXX set this to true once health checks pass?
	connection.Unlock()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := connection.Ping()
			if err != nil {
				connection.cancel()
			}
		case <-s.ctx.Done():
			ln.Log(ctx, connection, ln.Action("server context finished"))
			s.removeConn(ctx, connection)
			connection.Close()

			return
		case <-ctx.Done():
			ln.Log(ctx, connection, ln.Action("client context finished"))
			s.removeConn(ctx, connection)
			connection.Close()

			return
		}
	}
}

// addConn adds a connection to the pool of backend connections.
func (s *Server) addConn(ctx context.Context, connection *Connection) {
	s.connlock.Lock()
	s.conns[connection.conn] = connection
	s.connlock.Unlock()

	var conns []*Connection

	connection.Lock()
	val, ok := s.domains.Get(connection.domain)
	connection.Unlock()
	if ok {
		conns, ok = val.([]*Connection)
		if !ok {
			conns = nil

			connection.Lock()
			s.domains.Remove(connection.domain)
			connection.Unlock()
		}
	}

	conns = append(conns, connection)

	connection.Lock()
	s.domains.Set(connection.domain, conns)
	connection.Unlock()
}

// removeConn removes a connection from pool of backend connections.
func (s *Server) removeConn(ctx context.Context, connection *Connection) {
	ctx = opname.With(ctx, "removeConn")
	s.connlock.Lock()
	delete(s.conns, connection.conn)
	s.connlock.Unlock()

	auth := connection.Auth

	var conns []*Connection

	val, ok := s.domains.Get(auth.Domain)
	if ok {
		conns, ok = val.([]*Connection)
		if !ok {
			ln.Error(ctx, ErrCantRemoveWhatDoesntExist, connection, ln.Info("looking up for disconnect removal"))

			return
		}
	}

	for i, cntn := range conns {
		if cntn.id == connection.id {
			conns[i] = conns[len(conns)-1]
			conns = conns[:len(conns)-1]
		}
	}

	if len(conns) != 0 {
		s.domains.Set(auth.Domain, conns)
	} else {
		s.domains.Remove(auth.Domain)
	}
}

// RoundTrip sends a HTTP request to a backend and then returns its response.
func (s *Server) RoundTrip(req *http.Request) (*http.Response, error) {
	var conns []*Connection
	ctx := req.Context()
	ctx = opname.With(ctx, "tun2.Server.RoundTrip")

	f := ln.F{
		"req_remote":         req.RemoteAddr,
		"req_host":           req.Host,
		"req_uri":            req.RequestURI,
		"req_method":         req.Method,
		"req_content_length": req.ContentLength,
	}

	val, ok := s.domains.Get(req.Host)
	if ok {
		conns, ok = val.([]*Connection)
		if !ok {
			ln.Error(ctx, ErrNoSuchBackend, f, ln.Action("no backend available"))

			return gen502Page(req), nil
		}
	}

	var goodConns []*Connection
	for _, conn := range conns {
		conn.Lock()
		if conn.usable {
			goodConns = append(goodConns, conn)
		}
		conn.Unlock()
	}

	if len(goodConns) == 0 {
		ln.Error(ctx, ErrNoSuchBackend, f, ln.Action("no good backends available"))

		return gen502Page(req), nil
	}

	c := goodConns[rand.Intn(len(goodConns))]

	resp, err := c.RoundTrip(req)
	if err != nil {
		ln.Error(ctx, err, c, f, ln.Action("connection roundtrip"))

		defer c.cancel()
		return nil, err
	}

	ln.Log(ctx, c, ln.Action("http traffic"), f, ln.F{
		"resp_status_code":    resp.StatusCode,
		"resp_content_length": resp.ContentLength,
	})

	return resp, nil
}

// Auth is the authentication info the client passes to the server.
type Auth struct {
	Token  string `json:"token"`
	Domain string `json:"domain"`
}
