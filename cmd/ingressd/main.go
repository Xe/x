package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"sync"

	proxyproto "github.com/pires/go-proxyproto"
	"within.website/x/internal"
)

var (
	httpPort    = flag.Int("http-port", 80, "HTTP forwarding port")
	httpsPort   = flag.Int("https-port", 443, "HTTPS forwarding port")
	httpTarget  = flag.String("http-target", "10.216.118.119:80", "target address for http traffic")
	httpsTarget = flag.String("https-target", "10.216.118.119:443", "target address for https traffic")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting up", "httpPort", *httpPort, "httpsPort", *httpsPort, "httpTarget", *httpTarget, "httpsTarget", *httpsTarget)

	s := &Server{}

	httpLn, err := net.Listen("tcp", fmt.Sprintf(":%d", *httpPort))
	if err != nil {
		log.Fatalf("can't listen to HTTP port %d: %v", *httpPort, err)
		return
	}

	httpsLn, err := net.Listen("tcp", fmt.Sprintf(":%d", *httpsPort))
	if err != nil {
		log.Fatalf("can't listen to HTTPS port %d: %v", *httpsPort, err)
		return
	}

	go s.Handle(httpsLn, *httpsTarget)
	s.Handle(httpLn, *httpTarget)
}

type Server struct{}

func (s *Server) Handle(l net.Listener, dest string) {
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

func (s *Server) HandleConn(conn net.Conn, dest string) {
	defer conn.Close()

	slog.Debug("dialing remote host", "remoteHost", dest)

	destConn, err := net.Dial("tcp", dest)
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
		slog.Debug("done copying from dest to src")
	}()
	go func() {
		slog.Debug("copying from src to dest")
		defer wg.Done()
		io.Copy(destConn, conn)
		// Signal peer that no more data is coming.
		destConn.(*net.TCPConn).CloseWrite()
		slog.Debug("done copying from src to dest")
	}()

	wg.Wait()
	slog.Debug("done")
}
