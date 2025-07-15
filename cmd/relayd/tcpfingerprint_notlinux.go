//go:build !linux && !freebsd

package main

import (
	"net"
	"net/http"
)

func GetTCPFingerprint(r *http.Request) *JA4T {
	return nil
}

func assignTCPFingerprint(conn net.Conn) (*JA4T, error) {
	// Not supported on macOS
	return &JA4T{}, nil
}
