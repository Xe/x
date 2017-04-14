package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"

	kcp "github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

// Server represents the server side of kcpd. It listens on KCP and emits TCP connections from KCP streams.
type Server struct {
	cfg *Config
}

// NewServer creates a new Server and validates config.
func NewServer(cfg *Config) (*Server, error) {
	if cfg.Mode != "server" {
		return nil, ErrBadConfig
	}

	if cfg.ServerBindAddr == "" && cfg.ServerAthemeURL == "" && cfg.ServerAllowListEndpoint == "" && cfg.ServerLocalIRCd == "" && cfg.ServerWEBIRCPassword == "" && cfg.ServerTLSCert == "" && cfg.ServerTLSKey == "" {
		return nil, ErrBadConfig
	}

	return &Server{cfg: cfg}, nil
}

// ListenAndServe blockingly listens on the UDP port and relays KCP streams to TCP sockets.
func (s *Server) ListenAndServe() error {
	l, err := kcp.Listen(s.cfg.ServerBindAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	log.Printf("listening on KCP: %v", l.Addr())

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) error {
	defer conn.Close()

	log.Printf("new client: %v", conn.RemoteAddr())

	cert, err := tls.LoadX509KeyPair(s.cfg.ServerTLSCert, s.cfg.ServerTLSKey)
	if err != nil {
		return err
	}

	tcfg := &tls.Config{
		InsecureSkipVerify: true, // XXX hack remove
		Certificates:       []tls.Certificate{cert},
	}

	tlsConn := tls.Server(conn, tcfg)
	defer tlsConn.Close()

	session, err := smux.Server(tlsConn, smux.DefaultConfig())
	if err != nil {
		return err
	}
	defer session.Close()

	for {
		cstream, err := session.AcceptStream()
		if err != nil {
			log.Printf("client at %s error: %v", conn.RemoteAddr(), err)
			return err
		}

		ircConn, err := net.Dial("tcp", s.cfg.ServerLocalIRCd)
		if err != nil {
			log.Printf("client at %s error: %v", conn.RemoteAddr(), err)
			return err
		}

		host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

		fmt.Fprintf(ircConn, "WEBIRC %s %s %s %s\r\n", s.cfg.ServerWEBIRCPassword, RandStringRunes(8), host, host)

		go copyConn(cstream, ircConn)
	}

	return nil
}
