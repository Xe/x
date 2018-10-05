package tun2

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Xe/ln"
	"github.com/Xe/ln/opname"
	kcp "github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

// Client connects to a remote tun2 server and sets up authentication before routing
// individual HTTP requests to discrete streams that are reverse proxied to the eventual
// backend.
type Client struct {
	cfg *ClientConfig
}

// ClientConfig configures client with settings that the user provides.
type ClientConfig struct {
	TLSConfig  *tls.Config
	ConnType   string
	ServerAddr string
	Token      string
	Domain     string
	BackendURL string

	// internal use only
	forceTCPClear bool
}

// NewClient constructs an instance of Client with a given ClientConfig.
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("tun2: client config needed")
	}

	c := &Client{
		cfg: cfg,
	}

	return c, nil
}

// Connect dials the remote server and negotiates a client session with its
// configured server address. This will then continuously proxy incoming HTTP
// requests to the backend HTTP server.
//
// This is a blocking function.
func (c *Client) Connect(ctx context.Context) error {
	ctx = opname.With(ctx, "tun2.Client.connect")
	return c.connect(ctx, c.cfg.ServerAddr)
}

func closeLater(ctx context.Context, clo io.Closer) {
	<-ctx.Done()
	clo.Close()
}

func (c *Client) connect(ctx context.Context, serverAddr string) error {
	target, err := url.Parse(c.cfg.BackendURL)
	if err != nil {
		return err
	}

	s := &http.Server{
		Handler: httputil.NewSingleHostReverseProxy(target),
	}
	go closeLater(ctx, s)

	f := ln.F{
		"server_addr": serverAddr,
		"conn_type":   c.cfg.ConnType,
	}

	var conn net.Conn

	switch c.cfg.ConnType {
	case "tcp":
		if c.cfg.forceTCPClear {
			ln.Log(ctx, f, ln.Info("connecting over plain TCP"))
			conn, err = net.Dial("tcp", serverAddr)
		} else {
			conn, err = tls.Dial("tcp", serverAddr, c.cfg.TLSConfig)
		}

		if err != nil {
			return err
		}

	case "kcp":
		kc, err := kcp.Dial(serverAddr)
		if err != nil {
			return err
		}
		defer kc.Close()

		serverHost, _, _ := net.SplitHostPort(serverAddr)

		tc := c.cfg.TLSConfig.Clone()
		tc.ServerName = serverHost
		conn = tls.Client(kc, tc)
	}
	go closeLater(ctx, conn)

	ln.Log(ctx, f, ln.Info("connected"))

	session, err := smux.Client(conn, smux.DefaultConfig())
	if err != nil {
		return err
	}
	go closeLater(ctx, session)

	controlStream, err := session.AcceptStream()
	if err != nil {
		return err
	}
	go closeLater(ctx, controlStream)

	authData, err := json.Marshal(&Auth{
		Token:  c.cfg.Token,
		Domain: c.cfg.Domain,
	})
	if err != nil {
		return err
	}

	_, err = controlStream.Write(authData)
	if err != nil {
		return err
	}

	err = s.Serve(&smuxListener{
		conn:    conn,
		session: session,
	})
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		ln.Error(ctx, err, f, ln.Info("context error"))
	}
	return nil
}

// smuxListener wraps a smux session as a net.Listener.
type smuxListener struct {
	conn    net.Conn
	session *smux.Session
}

func (sl *smuxListener) Accept() (net.Conn, error) {
	return sl.session.AcceptStream()
}

func (sl *smuxListener) Addr() net.Addr {
	return sl.conn.LocalAddr()
}

func (sl *smuxListener) Close() error {
	return sl.session.Close()
}
