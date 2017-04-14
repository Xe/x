package main

import (
	"crypto/tls"
	"errors"
	"io"
	"net"

	kcp "github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

// Client opens a TCP listener and forwards traffic to the remote server over KCP.
type Client struct {
	cfg *Config

	listener net.Listener
}

// ErrBadConfig means the configuration is not correctly defined.
var ErrBadConfig = errors.New("kcpd: bad configuration file")

// NewClient constructs a new client with a given config.
func NewClient(cfg *Config) (*Client, error) {
	if cfg.Mode != "client" {
		return nil, ErrBadConfig
	}

	if cfg.ClientServerAddress == "" && cfg.ClientUsername == "" && cfg.ClientPassword == "" && cfg.ClientBindaddr == "" {
		return nil, ErrBadConfig
	}

	return &Client{cfg: cfg}, nil
}

// Dial blockingly connects to the remote server and relays TCP traffic.
func (c *Client) Dial() error {
	conn, err := kcp.Dial(c.cfg.ClientServerAddress)
	if err != nil {
		return err
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true, // XXX hack please remove
	})
	defer tlsConn.Close()

	session, err := smux.Client(tlsConn, smux.DefaultConfig())
	if err != nil {
		return err
	}
	defer session.Close()

	l, err := net.Listen("tcp", c.cfg.ClientBindaddr)
	if err != nil {
		return err
	}
	defer l.Close()
	c.listener = l

	for {
		cconn, err := l.Accept()
		if err != nil {
			break
		}

		cstream, err := session.OpenStream()
		if err != nil {
			break
		}

		go copyConn(cconn, cstream)
	}

	return nil
}

// Close frees resouces acquired in the client.
func (c *Client) Close() error {
	return c.listener.Close()
}

// copyConn copies one connection to another bidirectionally.
func copyConn(left, right net.Conn) error {
	defer left.Close()
	defer right.Close()

	go io.Copy(left, right)
	io.Copy(right, left)

	return nil
}
