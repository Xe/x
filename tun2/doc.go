/*
Package tun2 tunnels HTTP requests over existing, long-lived connections using
smux[1] and optionally kcp[2] to enable more reliable transport.

Currently this only works on a per-domain basis, but it is designed to be
flexible enough to support path-based routing as an addition in the future.

[1]: https://github.com/xtaci/smux
[2]: https://github.com/xtaci/kcp-go
*/
package tun2
