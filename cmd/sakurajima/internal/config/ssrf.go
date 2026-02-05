package config

import (
	"errors"
	"net"
	"net/url"
)

var ErrPrivateIP = errors.New("target points to private IP address (SSRF protection)")

// isPrivateIP checks if an IP address is in a private or reserved range.
// This includes loopback, private networks, link-local, CGNAT, and other reserved ranges.
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}

	// Convert to 4-byte representation for IPv4-mapped IPv6 addresses
	ip4 := ip.To4()
	if ip4 == nil {
		// It's a pure IPv6 address
		// Check for IPv6 unique local addresses (fc00::/7)
		if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
			return true
		}
		return false
	}

	// Check CGNAT range (100.64.0.0/10)
	if ip4[0] == 100 && (ip4[1]&0xc0) == 64 {
		return true
	}

	// Check TEST-NET ranges (192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24)
	if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 2 {
		return true
	}
	if ip4[0] == 198 && ip4[1] == 51 && ip4[2] == 100 {
		return true
	}
	if ip4[0] == 203 && ip4[1] == 0 && ip4[2] == 113 {
		return true
	}

	// Check multicast (224.0.0.0/4)
	if ip4[0]&0xf0 == 224 {
		return true
	}

	// Check reserved (240.0.0.0/4)
	if ip4[0]&0xf0 == 240 {
		return true
	}

	return false
}

// ValidateURLForSSRF checks if a URL points to a private IP address.
// This is a Server-Side Request Forgery (SSRF) protection measure.
//
// URLs with schemes other than http, https, or h2c are always allowed.
// URLs that use DNS names (not literal IPs) are allowed, but users should
// be aware of DNS rebinding attacks.
// Unix sockets are exempt as they are local by design.
func ValidateURLForSSRF(targetURL string) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	// Only validate http, https, and h2c schemes
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "h2c" {
		return nil
	}

	host := u.Hostname()
	if host == "" {
		return nil
	}

	// Check if the host is a literal IP address
	ip := net.ParseIP(host)
	if ip == nil {
		// It's a DNS name, not a literal IP
		// We allow DNS names, but users should be aware of DNS rebinding attacks
		return nil
	}

	// Check if the IP is in a private range
	if isPrivateIP(ip) {
		return ErrPrivateIP
	}

	return nil
}
