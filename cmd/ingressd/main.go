package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"sync"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"within.website/x/internal"
)

var (
	grpcPort    = flag.Int("grpc-port", 3932, "gRPC port for introspection and health checking")
	httpPort    = flag.Int("http-port", 80, "HTTP forwarding port")
	httpsPort   = flag.Int("https-port", 443, "HTTPS forwarding port")
	httpTarget  = flag.String("http-target", "10.216.118.119:80", "target address for http traffic")
	httpsTarget = flag.String("https-target", "10.216.118.119:443", "target address for https traffic")

	healthSrv = health.NewServer()
)

func main() {
	internal.HandleStartup()

	slog.Info("starting up", "httpPort", *httpPort, "httpsPort", *httpsPort, "httpTarget", *httpTarget, "httpsTarget", *httpsTarget)

	s := &Server{}

	grpcLn, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("can't listen to gRPC port %d: %v", *grpcPort, err)
	}

	httpLn, err := net.Listen("tcp", fmt.Sprintf(":%d", *httpPort))
	if err != nil {
		log.Fatalf("can't listen to HTTP port %d: %v", *httpPort, err)
	}

	httpsLn, err := net.Listen("tcp", fmt.Sprintf(":%d", *httpsPort))
	if err != nil {
		log.Fatalf("can't listen to HTTPS port %d: %v", *httpsPort, err)
	}

	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	reflection.Register(srv)

	healthSrv.SetServingStatus("grpc", healthpb.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("http", healthpb.HealthCheckResponse_NOT_SERVING)
	healthSrv.SetServingStatus("https", healthpb.HealthCheckResponse_NOT_SERVING)

	errCh := make(chan error, 3)
	go func() { errCh <- s.Handle(httpsLn, *httpsTarget, "https") }()
	go func() { errCh <- s.Handle(httpLn, *httpTarget, "http") }()
	go func() { errCh <- srv.Serve(grpcLn) }()

	go s.Probe(*httpTarget, "http")
	go s.Probe(*httpsTarget, "https")

	if err := <-errCh; err != nil {
		log.Fatal(err)
	}
}

type Server struct{}

const (
	probeInterval = 10 * time.Second
	probeTimeout  = 3 * time.Second
)

func (s *Server) Handle(l net.Listener, dest, service string) error {
	defer l.Close()

	backoff := 100 * time.Millisecond
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return err
			}

			slog.Error("can't accept connection; retrying",
				"localAddr", l.Addr().String(),
				"service", service,
				"err", err,
				"retryIn", backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, 5*time.Second)
			continue
		}
		backoff = 100 * time.Millisecond

		slog.Debug("accepted connection", "service", service, "addr", conn.RemoteAddr().String())
		go s.HandleConn(conn, dest)
	}
}

// Probe dials dest every probeInterval and mirrors its reachability into the
// gRPC health status for service. It logs only on status transitions.
func (s *Server) Probe(dest, service string) {
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()

	status := healthpb.HealthCheckResponse_NOT_SERVING
	for {
		conn, err := net.DialTimeout("tcp", dest, probeTimeout)
		if err != nil {
			if status == healthpb.HealthCheckResponse_SERVING {
				slog.Error("downstream probe failed", "service", service, "dest", dest, "err", err)
			}
			status = healthpb.HealthCheckResponse_NOT_SERVING
		} else {
			conn.Close()
			if status != healthpb.HealthCheckResponse_SERVING {
				slog.Info("downstream probe recovered", "service", service, "dest", dest)
			}
			status = healthpb.HealthCheckResponse_SERVING
		}
		healthSrv.SetServingStatus(service, status)

		<-ticker.C
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

	header := proxyproto.HeaderProxyFromAddrs(2, conn.RemoteAddr(), conn.LocalAddr())
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
		closeWrite(conn)
		slog.Debug("done copying from dest to src")
	}()
	go func() {
		slog.Debug("copying from src to dest")
		defer wg.Done()
		io.Copy(destConn, conn)
		// Signal peer that no more data is coming.
		closeWrite(destConn)
		slog.Debug("done copying from src to dest")
	}()

	wg.Wait()
	slog.Debug("done")
}

// closeWrite half-closes conn when it supports it. net.TCPConn does; test
// doubles and future wrappers may not.
func closeWrite(conn net.Conn) {
	cw, ok := conn.(interface{ CloseWrite() error })
	if !ok {
		return
	}
	if err := cw.CloseWrite(); err != nil {
		slog.Debug("can't half-close connection", "err", err)
	}
}
