package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/netip"
	"sync"

	proxyproto "github.com/pires/go-proxyproto"
	"within.website/x/internal"
)

var (
	httpPort          = flag.Int("http-port", 80, "HTTP forwarding port")
	httpsPort         = flag.Int("https-port", 443, "HTTPS forwarding port")
	nodeIPHTTPTarget  = flag.String("nodeip-http-target", "100.83.230.34:32677", "NodeIP HTTP target")
	nodeIPHTTPSTarget = flag.String("nodeip-https-target", "100.83.230.34:32537", "NodeIP HTTPS target")
	ipv4Subnet        = flag.String("ipv4-subnet", "10.255.255.0/24", "IPv4 private subnet for the userspace WireGuard network")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting up", "httpPort", *httpPort, "httpsPort", *httpsPort, "nodeIPHTTPTarget", *nodeIPHTTPTarget, "nodeIPHTTPSTarget", *nodeIPHTTPSTarget)

	ul, err := net.Listen("tcp", fmt.Sprintf(":%d", *httpPort))
	if err != nil {
		log.Fatalf("can't listen to HTTP port %d: %v", *httpPort, err)
		return
	}

	s := &Server{
		d: &net.Dialer{},
	}

	s.Handle(ul, netip.MustParseAddrPort(*nodeIPHTTPTarget))
}

type Server struct {
	d *net.Dialer
}

func (s *Server) Handle(l net.Listener, dest netip.AddrPort) {
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			slog.Error("can't accept connection", "localAddr", l.Addr().String(), "err", err)
			return
		}

		slog.Debug("accepted connection", "addr", conn.RemoteAddr().String())

		go s.HandleConn(conn, dest)
	}
}

func (s *Server) HandleConn(conn net.Conn, dest netip.AddrPort) {
	defer conn.Close()

	slog.Debug("dialing remote host", "remoteHost", dest.String())

	destConn, err := s.d.Dial("tcp", dest.String())
	if err != nil {
		slog.Error("can't dial downstream", "err", err)
		return
	}
	defer destConn.Close()

	slog.Debug("dialed remote host", "remoteAddr", conn.RemoteAddr().String(), "destAddr", destConn.RemoteAddr().String())

	header := proxyproto.HeaderProxyFromAddrs(2, conn.RemoteAddr(), destConn.RemoteAddr())
	if _, err := header.WriteTo(destConn); err != nil {
		slog.Error("can't write haproxy header to downstream", "header", header, "err", err)
		return
	}

	slog.Debug("wrote proxy header", "header", header)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		slog.Debug("copying from dest to src")
		defer wg.Done()
		io.Copy(conn, destConn)
		// Signal peer that no more data is coming.
		conn.(*net.TCPConn).CloseWrite()
	}()
	go func() {
		slog.Debug("copying from src to dest")
		defer wg.Done()
		io.Copy(destConn, conn)
		// Signal peer that no more data is coming.
		destConn.(*net.TCPConn).CloseWrite()
	}()

	wg.Wait()
	slog.Debug("done")
}
