package main

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
)

func applyTLSFingerprinter(server *http.Server) {
	server.TLSConfig = server.TLSConfig.Clone()

	getCertificate := server.TLSConfig.GetCertificate
	if getCertificate == nil {
		server.TLSConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			ja3n, ja4 := buildTLSFingerprint(clientHello)
			ptr := clientHello.Context().Value(tlsFingerprintKey{})
			if fpPtr, ok := ptr.(*TLSFingerprint); ok && ptr != nil && fpPtr != nil {
				fpPtr.ja3n.Store(&ja3n)
				fpPtr.ja4.Store(&ja4)
			}

			return nil, nil
		}
	} else {
		server.TLSConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			ja3n, ja4 := buildTLSFingerprint(clientHello)
			ptr := clientHello.Context().Value(tlsFingerprintKey{})
			if fpPtr, ok := ptr.(*TLSFingerprint); ok && ptr != nil && fpPtr != nil {
				fpPtr.ja3n.Store(&ja3n)
				fpPtr.ja4.Store(&ja4)
			}

			return getCertificate(clientHello)
		}
	}
	server.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, tlsFingerprintKey{}, &TLSFingerprint{})
	}
}

type tlsFingerprintKey struct{}
type TLSFingerprint struct {
	ja3n atomic.Pointer[TLSFingerprintJA3N]
	ja4  atomic.Pointer[TLSFingerprintJA4]
}

type TLSFingerprintJA3N [md5.Size]byte

func (f TLSFingerprintJA3N) String() string {
	return hex.EncodeToString(f[:])
}

type TLSFingerprintJA4 struct {
	A [10]byte
	B [6]byte
	C [6]byte
}

func (f TLSFingerprintJA4) String() string {
	return strings.Join([]string{
		string(f.A[:]),
		hex.EncodeToString(f.B[:]),
		hex.EncodeToString(f.C[:]),
	}, "_")
}

func (f *TLSFingerprint) JA3N() *TLSFingerprintJA3N {
	return f.ja3n.Load()
}

func (f *TLSFingerprint) JA4() *TLSFingerprintJA4 {
	return f.ja4.Load()
}

const greaseMask = 0x0F0F
const greaseValue = 0x0a0a

// TLS extension numbers
const (
	extensionServerName              uint16 = 0
	extensionStatusRequest           uint16 = 5
	extensionSupportedCurves         uint16 = 10 // supported_groups in TLS 1.3, see RFC 8446, Section 4.2.7
	extensionSupportedPoints         uint16 = 11
	extensionSignatureAlgorithms     uint16 = 13
	extensionALPN                    uint16 = 16
	extensionSCT                     uint16 = 18
	extensionExtendedMasterSecret    uint16 = 23
	extensionSessionTicket           uint16 = 35
	extensionPreSharedKey            uint16 = 41
	extensionEarlyData               uint16 = 42
	extensionSupportedVersions       uint16 = 43
	extensionCookie                  uint16 = 44
	extensionPSKModes                uint16 = 45
	extensionCertificateAuthorities  uint16 = 47
	extensionSignatureAlgorithmsCert uint16 = 50
	extensionKeyShare                uint16 = 51
	extensionQUICTransportParameters uint16 = 57
	extensionRenegotiationInfo       uint16 = 0xff01
	extensionECHOuterExtensions      uint16 = 0xfd00
	extensionEncryptedClientHello    uint16 = 0xfe0d
)

func tlsFingerprintJA3(hello *tls.ClientHelloInfo, sortExtensions bool) []byte {
	buf := make([]byte, 0, 256)

	{
		var sslVersion uint16
		var hasGrease bool
		for _, v := range hello.SupportedVersions {
			if v&greaseMask != greaseValue {
				if v > sslVersion {
					sslVersion = v
				}
			} else {
				hasGrease = true
			}
		}

		// maximum TLS 1.2 as specified on JA3, as TLS 1.3 is put in SupportedVersions
		if slices.Contains(hello.Extensions, extensionSupportedVersions) && hasGrease && sslVersion > tls.VersionTLS12 {
			sslVersion = tls.VersionTLS12
		}

		buf = strconv.AppendUint(buf, uint64(sslVersion), 10)
		buf = append(buf, ',')
	}

	n := 0
	for _, cipher := range hello.CipherSuites {
		//if !slices.Contains(greaseValues[:], cipher) {
		if cipher&greaseMask != greaseValue {
			buf = strconv.AppendUint(buf, uint64(cipher), 10)
			buf = append(buf, '-')
			n = 1
		}
	}

	buf = buf[:len(buf)-n]
	buf = append(buf, ',')
	n = 0

	extensions := hello.Extensions
	if sortExtensions {
		extensions = slices.Clone(extensions)
		slices.Sort(extensions)
	}

	for _, extension := range extensions {
		if extension&greaseMask != greaseValue {
			buf = strconv.AppendUint(buf, uint64(extension), 10)
			buf = append(buf, '-')
			n = 1
		}
	}

	buf = buf[:len(buf)-n]
	buf = append(buf, ',')
	n = 0

	for _, curve := range hello.SupportedCurves {
		if curve&greaseMask != greaseValue {
			buf = strconv.AppendUint(buf, uint64(curve), 10)
			buf = append(buf, '-')
			n = 1
		}
	}

	buf = buf[:len(buf)-n]
	buf = append(buf, ',')
	n = 0

	for _, point := range hello.SupportedPoints {
		buf = strconv.AppendUint(buf, uint64(point), 10)
		buf = append(buf, '-')
		n = 1
	}

	buf = buf[:len(buf)-n]

	sum := md5.Sum(buf)
	return sum[:]
}

func tlsFingerprintJA4(hello *tls.ClientHelloInfo) (ja4 TLSFingerprintJA4) {
	buf := make([]byte, 0, 10)

	// TODO: t = TLS, q = QUIC
	buf = append(buf, 't')

	{
		var sslVersion uint16
		for _, v := range hello.SupportedVersions {
			if v&greaseMask != greaseValue {
				if v > sslVersion {
					sslVersion = v
				}
			}
		}

		switch sslVersion {
		case tls.VersionSSL30:
			buf = append(buf, 's', '3')
		case tls.VersionTLS10:
			buf = append(buf, '1', '0')
		case tls.VersionTLS11:
			buf = append(buf, '1', '1')
		case tls.VersionTLS12:
			buf = append(buf, '1', '2')
		case tls.VersionTLS13:
			buf = append(buf, '1', '3')
		default:
			sslVersion -= 0x0201
			buf = strconv.AppendUint(buf, uint64(sslVersion>>8), 10)
			buf = strconv.AppendUint(buf, uint64(sslVersion&0xff), 10)
		}

	}

	if slices.Contains(hello.Extensions, extensionServerName) && hello.ServerName != "" {
		buf = append(buf, 'd')
	} else {
		buf = append(buf, 'i')
	}

	ciphers := make([]uint16, 0, len(hello.CipherSuites))
	for _, cipher := range hello.CipherSuites {
		if cipher&greaseMask != greaseValue {
			ciphers = append(ciphers, cipher)
		}
	}

	extensionCount := 0
	extensions := make([]uint16, 0, len(hello.Extensions))
	for _, extension := range hello.Extensions {
		if extension&greaseMask != greaseValue {
			extensionCount++
			if extension != extensionALPN && extension != extensionServerName {
				extensions = append(extensions, extension)
			}
		}
	}

	schemes := make([]tls.SignatureScheme, 0, len(hello.SignatureSchemes))

	for _, scheme := range hello.SignatureSchemes {
		if scheme&greaseMask != greaseValue {
			schemes = append(schemes, scheme)
		}
	}

	//TODO: maybe little endian
	slices.Sort(ciphers)
	slices.Sort(extensions)
	//slices.Sort(schemes)

	if len(ciphers) < 10 {
		buf = append(buf, '0')
		buf = strconv.AppendUint(buf, uint64(len(ciphers)), 10)
	} else if len(ciphers) > 99 {
		buf = append(buf, '9', '9')
	} else {
		buf = strconv.AppendUint(buf, uint64(len(ciphers)), 10)
	}

	if extensionCount < 10 {
		buf = append(buf, '0')
		buf = strconv.AppendUint(buf, uint64(extensionCount), 10)
	} else if extensionCount > 99 {
		buf = append(buf, '9', '9')
	} else {
		buf = strconv.AppendUint(buf, uint64(extensionCount), 10)
	}

	if len(hello.SupportedProtos) > 0 && len(hello.SupportedProtos[0]) > 1 {
		buf = append(buf, hello.SupportedProtos[0][0], hello.SupportedProtos[0][len(hello.SupportedProtos[0])-1])
	} else {
		buf = append(buf, '0', '0')
	}

	copy(ja4.A[:], buf)

	ja4.B = ja4SHA256(uint16SliceToHex(ciphers))

	extBuf := uint16SliceToHex(extensions)

	if len(schemes) > 0 {
		extBuf = append(extBuf, '_')
		extBuf = append(extBuf, uint16SliceToHex(schemes)...)
	}

	ja4.C = ja4SHA256(extBuf)

	return ja4
}

func uint16SliceToHex[T ~uint16](in []T) (out []byte) {
	if len(in) == 0 {
		return out
	}
	out = slices.Grow(out, hex.EncodedLen(len(in)*2)+len(in))

	for _, n := range in {
		out = append(out, fmt.Sprintf("%04x", uint16(n))...)
		out = append(out, ',')
	}
	out = out[:len(out)-1]

	return out
}

func ja4SHA256(buf []byte) [6]byte {
	if len(buf) == 0 {
		return [6]byte{0, 0, 0, 0, 0, 0}
	}
	sum := sha256.Sum256(buf)

	return [6]byte(sum[:6])
}

func buildTLSFingerprint(hello *tls.ClientHelloInfo) (ja3n TLSFingerprintJA3N, ja4 TLSFingerprintJA4) {
	return TLSFingerprintJA3N(tlsFingerprintJA3(hello, true)), tlsFingerprintJA4(hello)
}

func GetTLSFingerprint(r *http.Request) *TLSFingerprint {
	ptr := r.Context().Value(tlsFingerprintKey{})
	if fpPtr, ok := ptr.(*TLSFingerprint); ok && ptr != nil && fpPtr != nil {
		return fpPtr
	}
	return nil
}
