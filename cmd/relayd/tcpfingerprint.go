package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/mikioh/tcp"
	"github.com/mikioh/tcpinfo"
)

func assignTCPFingerprint(c net.Conn) (*JA4T, error) {
	tc, err := tcp.NewConn(c)
	if err != nil {
		return nil, err
	}

	var o tcpinfo.Info
	var b [256]byte
	i, err := tc.Option(o.Level(), o.Name(), b[:])
	if err != nil {
		return nil, err
	}

	ci, ok := i.(*tcpinfo.Info)
	if !ok {
		return nil, fmt.Errorf("can't make %T into *tcpinfo.Info", i)
	}

	result := &JA4T{
		Window: uint32(ci.Sys.SenderWindow),
		MSS:    uint16(ci.SenderMSS),
	}

	for _, opt := range ci.PeerOptions {
		switch opt.(type) {
		case tcpinfo.WindowScale:
			result.Options = append(result.Options, 3, uint8(opt.(tcpinfo.WindowScale)))
			result.WindowScale = uint8(opt.(tcpinfo.WindowScale))
		case tcpinfo.SACKPermitted:
			result.Options = append(result.Options, 4, 1)
		case tcpinfo.Timestamps:
			result.Options = append(result.Options, 8, 1)
		}
	}

	return result, nil
}

type tcpFingerprintKey struct{}

func GetTCPFingerprint(r *http.Request) *JA4T {
	ptr := r.Context().Value(tcpFingerprintKey{})
	if fpPtr, ok := ptr.(*JA4T); ok && ptr != nil && fpPtr != nil {
		return fpPtr
	}
	return nil
}

type JA4T struct {
	Window      uint32
	Options     []uint8
	MSS         uint16
	WindowScale uint8
}

func (j JA4T) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d_", j.Window)

	for i, opt := range j.Options {
		fmt.Fprint(&sb, opt)
		if i != len(j.Options)-1 {
			fmt.Fprint(&sb, "-")
		}
	}
	fmt.Fprint(&sb, "_")
	fmt.Fprintf(&sb, "%d_%d", j.MSS, j.WindowScale)

	return sb.String()
}
