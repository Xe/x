# ingressd bug fixes

## Context

`cmd/ingressd/main.go` is a small TCP ingress: it accepts on :80/:443, dials a downstream host, writes an HAProxy PROXY v2 header, and bidirectionally copies bytes; a gRPC health/reflection server listens on :3932. Review found five defects: the accept loop exits permanently on any transient `Accept` error (the gRPC server keeps the process alive, so systemd never restarts it); per-connection dial failures flip the health status to `SERVICE_UNKNOWN` one-way with last-writer-wins races; the PROXY v2 destination address is the backend's address instead of the original destination; two unchecked `*net.TCPConn` assertions can panic the process; and the gRPC listen-failure message prints the wrong port. Fix all five, drive health from a background reachability prober (TCP dial every 10s per target), and add regression tests.

## Approach

All edits are in `cmd/ingressd/main.go` except the new test file. Implement in this order; the tree builds after each step.

### 1. Fix the gRPC listen error message and dead code

In `main()`, change the gRPC listen failure to print `*grpcPort` instead of `*httpPort`:

```go
grpcLn, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
if err != nil {
	log.Fatalf("can't listen to gRPC port %d: %v", *grpcPort, err)
}
```

Delete the three unreachable `return` statements that follow the `log.Fatalf` calls for `grpcLn`, `httpLn`, and `httpsLn` (lines 40, 46, 52). `log.Fatalf` calls `os.Exit(1)`.

### 2. Make the accept loop survive transient errors

Change `Handle` to return an error and retry non-fatal `Accept` errors with capped exponential backoff. Replace the whole function (lines 69-85) with:

```go
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
```

Notes:
- Remove the `healthSrv.SetServingStatus(service, ...SERVING)` call from `Handle`; the prober (step 4) owns http/https status.
- `HandleConn` no longer needs its `service` parameter (after step 4 removes its only uses), so it becomes `func (s *Server) HandleConn(conn net.Conn, dest string)` and the call above passes only `conn, dest`.
- Add imports `errors` and `time`.

### 3. Run both accept loops and the gRPC server under one error channel

Replace the tail of `main()` (lines 55-64) with:

```go
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
```

This removes the old `go s.Handle(...)` calls, the old `healthSrv.SetServingStatus("grpc", ...)` call, and `log.Fatal(srv.Serve(grpcLn))`. If any listener dies unexpectedly, the process exits and systemd restarts it (`Restart=always` in `ingressd.service`).

### 4. Add a background reachability prober for http/https health

Add package-level constants and a `Probe` method:

```go
const (
	probeInterval = 10 * time.Second
	probeTimeout  = 3 * time.Second
)

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
```

Then delete the two per-connection health mutations in `HandleConn`:
- remove `healthSrv.SetServingStatus(service, healthpb.HealthCheckResponse_SERVICE_UNKNOWN)` from the `net.Dial` failure branch;
- remove the same call from the proxy-header write failure branch.

Keep both `slog.Error` calls. No other code reads `service` in `HandleConn`, hence the parameter removal in step 2.

### 5. Make the half-close type assertions panic-free

Add:

```go
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
```

Replace both `conn.(*net.TCPConn).CloseWrite()` and `destConn.(*net.TCPConn).CloseWrite()` with `closeWrite(conn)` / `closeWrite(destConn)`. Leave the `io.Copy` calls and the half-close ordering unchanged (it is the correct pattern; adding read/write deadlines would break long-lived connections such as websockets, so do not add them).

### 6. Send the original destination address in the PROXY header

In `HandleConn`, change:

```go
header := proxyproto.HeaderProxyFromAddrs(2, conn.RemoteAddr(), destConn.RemoteAddr())
```

to:

```go
header := proxyproto.HeaderProxyFromAddrs(2, conn.RemoteAddr(), conn.LocalAddr())
```

PROXY v2's destination field is the original destination of the client connection (the accepting socket's local address), not the backend's address. `conn.LocalAddr()` is the concrete `*net.TCPAddr` the client reached (e.g. public IP :80); `destConn.RemoteAddr()` made the backend report its own address. Keep the header write and its error log unchanged.

### 7. Add regression tests

Create `cmd/ingressd/main_test.go` (`package main`) with exactly these two tests:

```go
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
	go func() { errCh <- s.Handle(l, "127.0.0.1:1", "http") }()

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
```

The first test fails on the pre-fix code (old `Handle` returns after the first error and takes no error return), and pins the retry-then-recover behavior. The second pins the panic-free half-close path.

## Critical files & anchors

- `cmd/ingressd/main.go` — all production edits. `main()` lines 30-65 (listener setup, goroutine lifecycle, initial health statuses); `Handle` 69-85 (rewritten); `HandleConn` 87-133 (health calls removed, `service` param removed, header destination, `closeWrite`); new `Probe` method and constants.
- `cmd/ingressd/main_test.go` — new file, exact contents above.
- `cmd/ingressd/ingressd.service` — no change; read-only context confirming `Restart=always` is the restart mechanism relied on by step 3.
- `cmd/ingressd/yeetfile.js` — no change; it builds both arches and installs the unit.

## Verification

Run from the repo root `/home/xe/Code/Xe/x` (verified installed here: `go`, `grpcurl` 1.9.4, `socat`, `xxd`, `nc`).

1. Static + unit:

```bash
go build ./cmd/ingressd
go vet ./cmd/ingressd
go test -race ./cmd/ingressd
```

`go test -race` must pass both new tests and report no races. `TestHandleRetriesAcceptErrors` proves step 2 (3 Accept calls, returns only `net.ErrClosed`); `TestCloseWriteNonTCPConn` proves step 5.

2. End-to-end health prober and PROXY header:

```bash
mkdir -p /tmp/ingressd-smoke
rm -f /tmp/ingressd-smoke/capture.bin /tmp/ingressd-smoke/capture-https.bin
go build -o /tmp/ingressd-smoke/ingressd ./cmd/ingressd

# HTTP backend captures raw bytes to a file.
socat -u TCP-LISTEN:19000,reuseaddr,fork OPEN:/tmp/ingressd-smoke/capture.bin,creat,append &
SOCAT_HTTP=$!

# Daemon on unprivileged ports; HTTPS backend (19443) deliberately not running yet.
/tmp/ingressd-smoke/ingressd -grpc-port 13932 -http-port 18080 -https-port 18443 \
  -http-target 127.0.0.1:19000 -https-target 127.0.0.1:19443 \
  >/tmp/ingressd-smoke/daemon.log 2>&1 &
DAEMON=$!
sleep 1

# HTTP backend reachable -> SERVING; HTTPS backend down -> NOT_SERVING.
grpcurl -plaintext -d '{"service":"http"}'  127.0.0.1:13932 grpc.health.v1.Health/Check
grpcurl -plaintext -d '{"service":"https"}' 127.0.0.1:13932 grpc.health.v1.Health/Check
```

Expected: `{"status":"SERVING"}` then `{"status":"NOT_SERVING"}`.

```bash
# One proxied request; header + payload land in capture.bin.
printf 'hello' | timeout 2 socat - TCP:127.0.0.1:18080

xxd -p /tmp/ingressd-smoke/capture.bin | tr -d '\n' > /tmp/ingressd-smoke/capture.hex
grep -q '0d0a0d0a000d0a515549540a2111000c7f0000017f000001[0-9a-f]\{4\}46a068656c6c6f' /tmp/ingressd-smoke/capture.hex \
  && echo PROXY_HEADER_OK
```

Expected: `PROXY_HEADER_OK`. This asserts, in order: v2 signature, `0x21` PROXY command, `0x11` TCPv4, header length `0x000c`, source `127.0.0.1`, destination `127.0.0.1` (step 6; pre-fix the destination IP is identical but the destination port would be `4a38`/19000, so this check fails pre-fix), destination port `0x46a0` (18080), payload `hello`.

```bash
# Now bring the HTTPS backend up; the prober must flip the status within one interval.
socat -u TCP-LISTEN:19443,reuseaddr,fork OPEN:/tmp/ingressd-smoke/capture-https.bin,creat,append &
SOCAT_HTTPS=$!
sleep 11
grpcurl -plaintext -d '{"service":"https"}' 127.0.0.1:13932 grpc.health.v1.Health/Check
grep -q 'downstream probe recovered.*service=https' /tmp/ingressd-smoke/daemon.log && echo HEALTH_RECOVERED
```

Expected: `{"status":"SERVING"}` and `HEALTH_RECOVERED`. This proves the prober both detects failure and self-heals, replacing the old one-way per-connection status flip.

3. Cleanup:

```bash
kill "$DAEMON" "$SOCAT_HTTP" "$SOCAT_HTTPS"
rm -rf /tmp/ingressd-smoke
```

## Assumptions & contingencies

- Probe cadence is fixed at 10s with a 3s dial timeout, not exposed as flags; this is sufficient for a homelab ingress and avoids adding configuration surface. If `https` recovery in the smoke test takes longer than 11s on a loaded host, poll `grpcurl` every 2s up to 30s instead of changing the constants.
- Health service names stay `grpc`, `http`, `https`; no consumers outside this daemon were found in the repo.
- If the downstream ever rejects the new PROXY destination address, revert only the second argument of `HeaderProxyFromAddrs` to `destConn.RemoteAddr()`; nothing else depends on which address is used.
- The `accept: too many open files` string in the test is an ordinary error; only `net.ErrClosed` is treated as terminal. If a future listener wrapper returns a different terminal error, add it to the `errors.Is` check in `Handle`.
