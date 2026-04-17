# draft-meow-mrrp-00 Codec Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go package at `web/draft-meow-mrrp-00` that marshals and unmarshals frames defined by [draft-meow-mrrp-00](https://www.ietf.org/archive/id/draft-meow-mrrp-00.html), with literal cat-themed field names and automatic 1's-complement checksum handling.

**Architecture:** Pure codec — typed structs (`MeowHeader`, `MEowPseudoHeader`, three control messages) with `Marshal`/`Unmarshal` methods. Big-endian wire encoding. RFC 1071 1's-complement checksum computed automatically over pseudo-header + header + payload. No network I/O.

**Tech Stack:** Go (stdlib `encoding/binary` only), table-driven tests per `go-table-driven-tests` skill.

**Spec:** `docs/superpowers/specs/2026-04-17-draft-meow-mrrp-00-design.md`

---

## File Structure

All files live in `web/draft-meow-mrrp-00/`:

| File               | Responsibility                                                                                    |
| ------------------ | ------------------------------------------------------------------------------------------------- |
| `doc.go`           | Package doc comment: pointer to draft, TCP analogue note                                          |
| `mrrp.go`          | `MeowFlag` type and bit constants, exported `Err*` variables                                      |
| `checksum.go`      | `Checksum(parts ...[]byte) uint16` (RFC 1071 1's-complement)                                      |
| `pseudo.go`        | `MEowPseudoHeader` struct + `Marshal()` (12 bytes, big-endian)                                    |
| `header.go`        | `MeowHeader` struct + `Marshal(p MEowPseudoHeader)` and `Unmarshal(p MEowPseudoHeader, b []byte)` |
| `messages.go`      | `MeowMeowMeow`, `MeowMrrpMessage`, `MrowMeowMeowMeowMeow` types + their methods + `ParseMessage`  |
| `checksum_test.go` | Table-driven tests for `Checksum`                                                                 |
| `pseudo_test.go`   | Table-driven tests for pseudo-header marshal                                                      |
| `header_test.go`   | Table-driven tests for header round-trip + checksum validation + error cases                      |
| `messages_test.go` | Table-driven tests for control messages + `ParseMessage`                                          |

No per-package `LICENSE` — repo root LICENSE covers it (matches `web/useragent`, most siblings).

---

## Task 1: Package skeleton — doc.go and mrrp.go

**Files:**

- Create: `web/draft-meow-mrrp-00/doc.go`
- Create: `web/draft-meow-mrrp-00/mrrp.go`

- [ ] **Step 1: Create `doc.go`**

```go
// Package mrrp implements draft-meow-mrrp-00, an April 2026 IETF
// Internet-Draft whose wire format is structurally identical to TCP
// (RFC 9293) plus an IP-style pseudo-header for checksum computation.
//
// The draft also defines three ICMP-shaped control messages (types 0,
// 1, and 2). All field names in this package follow the draft's
// literal cat-themed naming; doc comments carry the TCP equivalent
// where applicable.
//
// This package is a pure codec — it parses bytes into typed structs
// and serializes typed structs back to bytes. It performs no network
// I/O.
//
// Reference: https://www.ietf.org/archive/id/draft-meow-mrrp-00.html
package mrrp
```

- [ ] **Step 2: Create `mrrp.go` with flag constants and errors**

```go
package mrrp

import "errors"

// MeowFlag is the 8-bit flag field of a MeowHeader. Bit positions
// follow the draft's left-to-right ordering: MEOW is the most
// significant bit.
type MeowFlag uint8

const (
	FlagMEOW  MeowFlag = 1 << 7 // "Meow Meow Meow"
	FlagNYA   MeowFlag = 1 << 6 // "MEOW-Mrrp"
	FlagMRRP  MeowFlag = 1 << 5 // "Urgent meowing mrrp mrow nyaaa"
	FlagMOEW  MeowFlag = 1 << 4 // "Mrrrreeaowwww meow meow meow"
	FlagMROW  MeowFlag = 1 << 3 // "Meow meow"
	FlagMIAU  MeowFlag = 1 << 2 // "Meow meow meow"
	FlagMIAOW MeowFlag = 1 << 1 // "Meow meow meow"
	FlagPURR  MeowFlag = 1 << 0 // "Meow meow meow meow meow"
)

// MeowHeaderMinSize is the minimum size of a MeowHeader on the wire,
// in bytes (no options, no payload).
const MeowHeaderMinSize = 20

// PseudoHeaderSize is the size of a MEowPseudoHeader on the wire, in
// bytes.
const PseudoHeaderSize = 12

var (
	ErrShortHeader        = errors.New("mrrp: short header")
	ErrBadDataOffset      = errors.New("mrrp: bad data offset")
	ErrOptionsLength      = errors.New("mrrp: options length exceeds input")
	ErrChecksumMismatch   = errors.New("mrrp: checksum mismatch")
	ErrUnknownMessageType = errors.New("mrrp: unknown message type")
	ErrShortMessage       = errors.New("mrrp: short message")
)
```

- [ ] **Step 3: Verify the package builds**

Run: `go build ./web/draft-meow-mrrp-00/...`
Expected: clean exit, no output.

- [ ] **Step 4: Commit**

```bash
git add web/draft-meow-mrrp-00/doc.go web/draft-meow-mrrp-00/mrrp.go
git commit --signoff -m "$(cat <<'EOF'
feat(mrrp): add package skeleton with flag constants and errors

Implements the foundation for draft-meow-mrrp-00: package doc,
MeowFlag bit constants in draft order, and exported error values.

Assisted-by: Claude Opus 4.7 via Claude Code
Reviewbot-request: yes
EOF
)"
```

---

## Task 2: Checksum — RFC 1071 1's-complement

**Files:**

- Create: `web/draft-meow-mrrp-00/checksum_test.go`
- Create: `web/draft-meow-mrrp-00/checksum.go`

- [ ] **Step 1: Write the failing test**

`web/draft-meow-mrrp-00/checksum_test.go`:

```go
package mrrp_test

import (
	"testing"

	mrrp "within.website/x/web/draft-meow-mrrp-00"
)

func TestChecksum(t *testing.T) {
	tests := []struct {
		name  string
		parts [][]byte
		want  uint16
	}{
		{
			name:  "empty",
			parts: nil,
			want:  0xFFFF,
		},
		{
			name:  "single zero byte",
			parts: [][]byte{{0x00}},
			want:  0xFFFF,
		},
		{
			// RFC 1071 §3 example: 00 01 f2 03 f4 f5 f6 f7
			// 1's-complement sum = ddf2, complement = 220d.
			name:  "rfc1071 example",
			parts: [][]byte{{0x00, 0x01, 0xf2, 0x03, 0xf4, 0xf5, 0xf6, 0xf7}},
			want:  0x220d,
		},
		{
			// Odd length: trailing byte is padded with a zero in the
			// low half of the 16-bit word.
			name:  "odd length",
			parts: [][]byte{{0x12, 0x34, 0x56}},
			want:  ^uint16(0x12 + 0x34<<8 + 0x56), // wrong, see step 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mrrp.Checksum(tt.parts...)
			if got != tt.want {
				t.Errorf("Checksum() = %#04x, want %#04x", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Compute the correct "odd length" expected value**

For input `{0x12, 0x34, 0x56}`:

- Word 1: `0x1234`
- Word 2: `0x5600` (trailing 0x56 in high byte, 0x00 padding in low byte — network byte order)
- Sum: `0x1234 + 0x5600 = 0x6834`
- 1's complement: `^0x6834 = 0x97cb`

Replace the `want` expression in the "odd length" case with `0x97cb`.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./web/draft-meow-mrrp-00/ -run TestChecksum -v`
Expected: FAIL — `Checksum` undefined.

- [ ] **Step 4: Implement `checksum.go`**

```go
package mrrp

// Checksum computes the 16-bit 1's-complement checksum (RFC 1071) of
// the concatenation of parts. When applied to MEOW frames, the caller
// passes the marshalled pseudo-header, the header bytes (with the
// Mrrrowww field zeroed), and the payload, in that order.
//
// Each part is treated as an independent byte stream concatenated end
// to end; the checksum is computed over the resulting stream as if
// it were a single buffer. A trailing odd byte is padded with a zero
// low byte to form the final 16-bit word.
func Checksum(parts ...[]byte) uint16 {
	var (
		sum  uint32
		half uint16
		have bool
	)
	for _, p := range parts {
		i := 0
		if have {
			half |= uint16(p[0])
			sum += uint32(half)
			i = 1
			have = false
		}
		for ; i+1 < len(p); i += 2 {
			sum += uint32(uint16(p[i])<<8 | uint16(p[i+1]))
		}
		if i < len(p) {
			half = uint16(p[i]) << 8
			have = true
		}
	}
	if have {
		sum += uint32(half)
	}
	for sum>>16 != 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return ^uint16(sum)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./web/draft-meow-mrrp-00/ -run TestChecksum -v`
Expected: PASS, all four subtests pass.

- [ ] **Step 6: Commit**

```bash
git add web/draft-meow-mrrp-00/checksum.go web/draft-meow-mrrp-00/checksum_test.go
git commit --signoff -m "$(cat <<'EOF'
feat(mrrp): add RFC 1071 1's-complement checksum

Implements Checksum(parts ...[]byte) uint16, used to compute and
validate the Mrrrowww field over a pseudo-header, header, and
payload.

Assisted-by: Claude Opus 4.7 via Claude Code
Reviewbot-request: yes
EOF
)"
```

---

## Task 3: Pseudo-header

**Files:**

- Create: `web/draft-meow-mrrp-00/pseudo_test.go`
- Create: `web/draft-meow-mrrp-00/pseudo.go`

- [ ] **Step 1: Write the failing test**

`web/draft-meow-mrrp-00/pseudo_test.go`:

```go
package mrrp_test

import (
	"bytes"
	"testing"

	mrrp "within.website/x/web/draft-meow-mrrp-00"
)

func TestMEowPseudoHeaderMarshal(t *testing.T) {
	tests := []struct {
		name string
		ph   mrrp.MEowPseudoHeader
		want []byte
	}{
		{
			name: "zero",
			ph:   mrrp.MEowPseudoHeader{},
			want: []byte{
				0, 0, 0, 0, // src
				0, 0, 0, 0, // dst
				0,    // miau
				0,    // MRRP
				0, 0, // length
			},
		},
		{
			name: "populated",
			ph: mrrp.MEowPseudoHeader{
				MeeowwNyaaaaa:      [4]byte{10, 0, 0, 1},
				PurrreeowwwMrrrrrp: [4]byte{10, 0, 0, 2},
				Miau:               0,
				MRRP:               0xCA,
				MEowMeeoww:         0x1234,
			},
			want: []byte{
				10, 0, 0, 1,
				10, 0, 0, 2,
				0,
				0xCA,
				0x12, 0x34,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ph.Marshal()
			if !bytes.Equal(got, tt.want) {
				t.Errorf("Marshal() = %x, want %x", got, tt.want)
			}
			if len(got) != mrrp.PseudoHeaderSize {
				t.Errorf("len(Marshal()) = %d, want %d", len(got), mrrp.PseudoHeaderSize)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./web/draft-meow-mrrp-00/ -run TestMEowPseudoHeaderMarshal -v`
Expected: FAIL — `MEowPseudoHeader` undefined.

- [ ] **Step 3: Implement `pseudo.go`**

```go
package mrrp

import "encoding/binary"

// MEowPseudoHeader is the pseudo-header used as input to the
// MeowHeader checksum computation. It is never transmitted on the
// wire; it exists solely so the checksum covers the source and
// destination addresses, mirroring the TCP/IP pseudo-header.
type MEowPseudoHeader struct {
	MeeowwNyaaaaa      [4]byte // source address
	PurrreeowwwMrrrrrp [4]byte // destination address
	Miau               uint8   // zero byte
	MRRP               uint8   // protocol byte
	MEowMeeoww         uint16  // length of MeowHeader + payload, in bytes
}

// Marshal returns the 12-byte big-endian wire encoding of p.
func (p MEowPseudoHeader) Marshal() []byte {
	b := make([]byte, PseudoHeaderSize)
	copy(b[0:4], p.MeeowwNyaaaaa[:])
	copy(b[4:8], p.PurrreeowwwMrrrrrp[:])
	b[8] = p.Miau
	b[9] = p.MRRP
	binary.BigEndian.PutUint16(b[10:12], p.MEowMeeoww)
	return b
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./web/draft-meow-mrrp-00/ -run TestMEowPseudoHeaderMarshal -v`
Expected: PASS, both subtests pass.

- [ ] **Step 5: Commit**

```bash
git add web/draft-meow-mrrp-00/pseudo.go web/draft-meow-mrrp-00/pseudo_test.go
git commit --signoff -m "$(cat <<'EOF'
feat(mrrp): add MEowPseudoHeader and its wire encoding

The pseudo-header is never transmitted; it exists only as input to
the MeowHeader checksum, mirroring the TCP/IP pseudo-header.

Assisted-by: Claude Opus 4.7 via Claude Code
Reviewbot-request: yes
EOF
)"
```

---

## Task 4: Main header — Marshal

**Files:**

- Create: `web/draft-meow-mrrp-00/header.go`
- Create: `web/draft-meow-mrrp-00/header_test.go`

- [ ] **Step 1: Write the failing test (Marshal half only)**

`web/draft-meow-mrrp-00/header_test.go`:

```go
package mrrp_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	mrrp "within.website/x/web/draft-meow-mrrp-00"
)

// minimalPseudo is a stand-in pseudo-header used by tests that don't
// care about specific addresses — only that the same pseudo-header is
// used for marshal and unmarshal.
func minimalPseudo(length uint16) mrrp.MEowPseudoHeader {
	return mrrp.MEowPseudoHeader{
		MeeowwNyaaaaa:      [4]byte{127, 0, 0, 1},
		PurrreeowwwMrrrrrp: [4]byte{127, 0, 0, 2},
		Miau:               0,
		MRRP:               0xCA,
		MEowMeeoww:         length,
	}
}

func TestMeowHeaderMarshal(t *testing.T) {
	tests := []struct {
		name    string
		h       mrrp.MeowHeader
		// wantPrefix is the marshalled bytes EXCEPT the checksum
		// (offsets 16-17), which we check separately.
		wantPrefix []byte
		wantSuffix []byte // bytes after the checksum field
	}{
		{
			name: "minimal header, no options, no payload",
			h: mrrp.MeowHeader{
				MeowMrrp:          0x1234,
				MoewMrrp:          0x5678,
				MeeeeowNyaaaa:     0xDEADBEEF,
				MrrrreeaowwwwMrrp: 0xCAFEBABE,
				MeowMrrrrp:        5,
				Purrr:             0,
				Flags:             mrrp.FlagMEOW | mrrp.FlagMRRP,
				Mraoww:            0x4000,
				UrgentMeowing:     0,
			},
			wantPrefix: []byte{
				0x12, 0x34, // MeowMrrp
				0x56, 0x78, // MoewMrrp
				0xDE, 0xAD, 0xBE, 0xEF, // MeeeeowNyaaaa
				0xCA, 0xFE, 0xBA, 0xBE, // MrrrreeaowwwwMrrp
				0x50,       // MeowMrrrrp=5 << 4 | Purrr=0
				0xA0,       // FlagMEOW (1<<7) | FlagMRRP (1<<5) = 0xA0
				0x40, 0x00, // Mraoww
			},
			wantSuffix: []byte{
				0x00, 0x00, // UrgentMeowing
			},
		},
		{
			name: "with options and payload",
			h: mrrp.MeowHeader{
				MeowMrrp:          1,
				MoewMrrp:          2,
				MeeeeowNyaaaa:     3,
				MrrrreeaowwwwMrrp: 4,
				MeowMrrrrp:        7, // 5 + 2 = 8 bytes of options
				Purrr:             0,
				Flags:             mrrp.FlagPURR,
				Mraoww:            0xFFFF,
				UrgentMeowing:     0xABCD,
				Meeeowo:           []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				Meow:              []byte("nyaa"),
			},
			wantPrefix: []byte{
				0x00, 0x01,
				0x00, 0x02,
				0x00, 0x00, 0x00, 0x03,
				0x00, 0x00, 0x00, 0x04,
				0x70, // 7 << 4
				0x01, // FlagPURR
				0xFF, 0xFF,
			},
			wantSuffix: append(
				append(
					[]byte{0xAB, 0xCD}, // UrgentMeowing
					0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, // options
				),
				[]byte("nyaa")...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ph := minimalPseudo(uint16(int(tt.h.MeowMrrrrp)*4 + len(tt.h.Meow)))
			got, err := tt.h.Marshal(ph)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			// Bytes 0-15 (prefix, before checksum)
			if !bytes.Equal(got[:16], tt.wantPrefix) {
				t.Errorf("prefix = %x, want %x", got[:16], tt.wantPrefix)
			}

			// Bytes 18+ (suffix, after checksum)
			if !bytes.Equal(got[18:], tt.wantSuffix) {
				t.Errorf("suffix = %x, want %x", got[18:], tt.wantSuffix)
			}

			// Checksum: zero out checksum field in `got`, recompute,
			// compare to the value Marshal wrote in.
			gotChecksum := binary.BigEndian.Uint16(got[16:18])
			zeroed := append([]byte(nil), got...)
			zeroed[16] = 0
			zeroed[17] = 0
			wantChecksum := mrrp.Checksum(ph.Marshal(), zeroed)
			if gotChecksum != wantChecksum {
				t.Errorf("checksum = %#04x, want %#04x", gotChecksum, wantChecksum)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./web/draft-meow-mrrp-00/ -run TestMeowHeaderMarshal -v`
Expected: FAIL — `MeowHeader` undefined.

- [ ] **Step 3: Implement `header.go` (Marshal only for now)**

```go
package mrrp

import "encoding/binary"

// MeowHeader is the main MEOW message header (Figure 1 of the
// draft). Its layout is structurally identical to the TCP header.
type MeowHeader struct {
	MeowMrrp          uint16   // src port equivalent
	MoewMrrp          uint16   // dst port equivalent
	MeeeeowNyaaaa     uint32   // sequence number
	MrrrreeaowwwwMrrp uint32   // acknowledgment number
	MeowMrrrrp        uint8    // 4 bits — data offset, in 32-bit words; min 5
	Purrr             uint8    // 4 bits — reserved, MUST be zero on send
	Flags             MeowFlag // 8 bits
	Mraoww            uint16   // window
	Mrrrowww          uint16   // checksum (filled in by Marshal)
	UrgentMeowing     uint16   // urgent pointer
	Meeeowo           []byte   // options, (MeowMrrrrp-5)*4 bytes; 32-bit aligned
	Meow              []byte   // payload
}

// Marshal returns the wire encoding of h, computing and writing the
// Mrrrowww checksum over the supplied pseudo-header, header, and
// payload (RFC 1071 1's-complement). The Mrrrowww field on h itself
// is not consulted.
//
// Marshal returns ErrBadDataOffset if h.MeowMrrrrp < 5 and
// ErrOptionsLength if len(h.Meeeowo) does not equal
// (h.MeowMrrrrp-5)*4 bytes.
func (h *MeowHeader) Marshal(p MEowPseudoHeader) ([]byte, error) {
	if h.MeowMrrrrp < 5 {
		return nil, ErrBadDataOffset
	}
	wantOptions := (int(h.MeowMrrrrp) - 5) * 4
	if len(h.Meeeowo) != wantOptions {
		return nil, ErrOptionsLength
	}

	headerLen := int(h.MeowMrrrrp) * 4
	out := make([]byte, headerLen+len(h.Meow))

	binary.BigEndian.PutUint16(out[0:2], h.MeowMrrp)
	binary.BigEndian.PutUint16(out[2:4], h.MoewMrrp)
	binary.BigEndian.PutUint32(out[4:8], h.MeeeeowNyaaaa)
	binary.BigEndian.PutUint32(out[8:12], h.MrrrreeaowwwwMrrp)
	out[12] = (h.MeowMrrrrp&0x0F)<<4 | (h.Purrr & 0x0F)
	out[13] = byte(h.Flags)
	binary.BigEndian.PutUint16(out[14:16], h.Mraoww)
	// out[16:18] checksum, left as zero for the computation
	binary.BigEndian.PutUint16(out[18:20], h.UrgentMeowing)
	copy(out[20:headerLen], h.Meeeowo)
	copy(out[headerLen:], h.Meow)

	cs := Checksum(p.Marshal(), out)
	binary.BigEndian.PutUint16(out[16:18], cs)
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./web/draft-meow-mrrp-00/ -run TestMeowHeaderMarshal -v`
Expected: PASS, both subtests pass.

- [ ] **Step 5: Commit**

```bash
git add web/draft-meow-mrrp-00/header.go web/draft-meow-mrrp-00/header_test.go
git commit --signoff -m "$(cat <<'EOF'
feat(mrrp): add MeowHeader.Marshal with checksum computation

Marshal serializes the 20+ byte header in big-endian, fills the
Mrrrowww checksum from the pseudo-header + header + payload, and
returns ErrBadDataOffset / ErrOptionsLength on malformed input.

Assisted-by: Claude Opus 4.7 via Claude Code
Reviewbot-request: yes
EOF
)"
```

---

## Task 5: Main header — Unmarshal + round-trip

**Files:**

- Modify: `web/draft-meow-mrrp-00/header.go`
- Modify: `web/draft-meow-mrrp-00/header_test.go`

- [ ] **Step 1: Add the failing test (round-trip + error cases)**

Append to `web/draft-meow-mrrp-00/header_test.go`:

```go
func TestMeowHeaderRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		h    mrrp.MeowHeader
	}{
		{
			name: "minimal",
			h: mrrp.MeowHeader{
				MeowMrrp:          1,
				MoewMrrp:          2,
				MeeeeowNyaaaa:     3,
				MrrrreeaowwwwMrrp: 4,
				MeowMrrrrp:        5,
				Flags:             mrrp.FlagMEOW,
				Mraoww:            0x1000,
			},
		},
		{
			name: "with options and payload",
			h: mrrp.MeowHeader{
				MeowMrrp:          0xAAAA,
				MoewMrrp:          0xBBBB,
				MeeeeowNyaaaa:     0x11223344,
				MrrrreeaowwwwMrrp: 0x55667788,
				MeowMrrrrp:        6,
				Flags:             mrrp.FlagMEOW | mrrp.FlagNYA | mrrp.FlagPURR,
				Mraoww:            0xCCCC,
				UrgentMeowing:     0xDDDD,
				Meeeowo:           []byte{0xDE, 0xAD, 0xBE, 0xEF},
				Meow:              []byte("mrrp mrrp mrrp"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ph := minimalPseudo(uint16(int(tt.h.MeowMrrrrp)*4 + len(tt.h.Meow)))
			b, err := tt.h.Marshal(ph)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got mrrp.MeowHeader
			if err := got.Unmarshal(ph, b); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			// Re-marshal and compare bytes — true round-trip check.
			b2, err := got.Marshal(ph)
			if err != nil {
				t.Fatalf("Marshal() of unmarshalled header: error = %v", err)
			}
			if !bytes.Equal(b, b2) {
				t.Errorf("round-trip mismatch:\n got %x\nwant %x", b2, b)
			}
		})
	}
}

func TestMeowHeaderUnmarshalErrors(t *testing.T) {
	// Build a valid frame to start from, then corrupt it.
	good := mrrp.MeowHeader{
		MeowMrrp:   1,
		MoewMrrp:   2,
		MeowMrrrrp: 5,
		Flags:      mrrp.FlagMEOW,
	}
	ph := minimalPseudo(20)
	b, err := good.Marshal(ph)
	if err != nil {
		t.Fatalf("setup Marshal error: %v", err)
	}

	tests := []struct {
		name string
		ph   mrrp.MEowPseudoHeader
		in   []byte
		want error
	}{
		{
			name: "short header",
			ph:   ph,
			in:   b[:10],
			want: mrrp.ErrShortHeader,
		},
		{
			name: "bad data offset (less than 5)",
			ph:   ph,
			in: func() []byte {
				c := append([]byte(nil), b...)
				c[12] = 4 << 4 // MeowMrrrrp = 4
				return c
			}(),
			want: mrrp.ErrBadDataOffset,
		},
		{
			name: "options length exceeds input",
			ph:   ph,
			in: func() []byte {
				c := append([]byte(nil), b...)
				c[12] = 10 << 4 // claims 40-byte header, only 20 supplied
				return c
			}(),
			want: mrrp.ErrOptionsLength,
		},
		{
			name: "checksum mismatch",
			ph:   ph,
			in: func() []byte {
				c := append([]byte(nil), b...)
				c[16] ^= 0xFF // corrupt checksum
				return c
			}(),
			want: mrrp.ErrChecksumMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h mrrp.MeowHeader
			err := h.Unmarshal(tt.ph, tt.in)
			if err != tt.want {
				t.Errorf("Unmarshal() error = %v, want %v", err, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./web/draft-meow-mrrp-00/ -run "TestMeowHeaderRoundTrip|TestMeowHeaderUnmarshalErrors" -v`
Expected: FAIL — `Unmarshal` undefined.

- [ ] **Step 3: Add `Unmarshal` to `header.go`**

Append to `web/draft-meow-mrrp-00/header.go`:

```go
// Unmarshal parses b into h, validating the Mrrrowww checksum against
// the supplied pseudo-header. Errors:
//
//   - ErrShortHeader if b is shorter than MeowHeaderMinSize.
//   - ErrBadDataOffset if the encoded MeowMrrrrp is less than 5.
//   - ErrOptionsLength if the encoded data offset implies a header
//     larger than b.
//   - ErrChecksumMismatch if the recomputed checksum does not match
//     the value carried in b.
//
// On success, h.Meeeowo aliases bytes inside b (no copy). Callers
// that mutate b after Unmarshal must copy first.
func (h *MeowHeader) Unmarshal(p MEowPseudoHeader, b []byte) error {
	if len(b) < MeowHeaderMinSize {
		return ErrShortHeader
	}

	dataOff := b[12] >> 4
	if dataOff < 5 {
		return ErrBadDataOffset
	}
	headerLen := int(dataOff) * 4
	if headerLen > len(b) {
		return ErrOptionsLength
	}

	// Validate checksum: zero the field in a scratch copy and
	// recompute over pseudo-header + header + payload.
	scratch := make([]byte, len(b))
	copy(scratch, b)
	scratch[16] = 0
	scratch[17] = 0
	if Checksum(p.Marshal(), scratch) != binary.BigEndian.Uint16(b[16:18]) {
		return ErrChecksumMismatch
	}

	h.MeowMrrp = binary.BigEndian.Uint16(b[0:2])
	h.MoewMrrp = binary.BigEndian.Uint16(b[2:4])
	h.MeeeeowNyaaaa = binary.BigEndian.Uint32(b[4:8])
	h.MrrrreeaowwwwMrrp = binary.BigEndian.Uint32(b[8:12])
	h.MeowMrrrrp = dataOff
	h.Purrr = b[12] & 0x0F
	h.Flags = MeowFlag(b[13])
	h.Mraoww = binary.BigEndian.Uint16(b[14:16])
	h.Mrrrowww = binary.BigEndian.Uint16(b[16:18])
	h.UrgentMeowing = binary.BigEndian.Uint16(b[18:20])
	if headerLen > MeowHeaderMinSize {
		h.Meeeowo = b[20:headerLen]
	} else {
		h.Meeeowo = nil
	}
	if len(b) > headerLen {
		h.Meow = b[headerLen:]
	} else {
		h.Meow = nil
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./web/draft-meow-mrrp-00/ -v`
Expected: PASS — all `TestMeowHeader*` tests, including the new round-trip and error cases.

- [ ] **Step 5: Commit**

```bash
git add web/draft-meow-mrrp-00/header.go web/draft-meow-mrrp-00/header_test.go
git commit --signoff -m "$(cat <<'EOF'
feat(mrrp): add MeowHeader.Unmarshal with checksum validation

Unmarshal parses the wire format, validates the Mrrrowww checksum
against the supplied pseudo-header, and returns specific errors for
short input, bad data offset, options overrun, and checksum
mismatch. Round-trip tests exercise minimal and option-bearing
frames.

Assisted-by: Claude Opus 4.7 via Claude Code
Reviewbot-request: yes
EOF
)"
```

---

## Task 6: Control messages (Type 0, 1, 2) and ParseMessage

**Files:**

- Create: `web/draft-meow-mrrp-00/messages.go`
- Create: `web/draft-meow-mrrp-00/messages_test.go`

- [ ] **Step 1: Write the failing test**

`web/draft-meow-mrrp-00/messages_test.go`:

```go
package mrrp_test

import (
	"bytes"
	"reflect"
	"testing"

	mrrp "within.website/x/web/draft-meow-mrrp-00"
)

func TestMessageMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		marshal func() []byte
		want    []byte
		parsed  any
	}{
		{
			name:    "type 0 MeowMeowMeow",
			marshal: func() []byte { return mrrp.MeowMeowMeow{}.Marshal() },
			want:    []byte{0x00},
			parsed:  mrrp.MeowMeowMeow{},
		},
		{
			name:    "type 1 MeowMrrpMessage",
			marshal: func() []byte { return mrrp.MeowMrrpMessage{}.Marshal() },
			want:    []byte{0x01},
			parsed:  mrrp.MeowMrrpMessage{},
		},
		{
			name: "type 2 MrowMeowMeowMeowMeow",
			marshal: func() []byte {
				return mrrp.MrowMeowMeowMeowMeow{Meeoww: 0xAB, MMN: 0x1234}.Marshal()
			},
			want:   []byte{0x02, 0xAB, 0x12, 0x34},
			parsed: mrrp.MrowMeowMeowMeowMeow{Meeoww: 0xAB, MMN: 0x1234},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.marshal()
			if !bytes.Equal(got, tt.want) {
				t.Errorf("Marshal() = %x, want %x", got, tt.want)
			}
			parsed, err := mrrp.ParseMessage(got)
			if err != nil {
				t.Fatalf("ParseMessage() error = %v", err)
			}
			if !reflect.DeepEqual(parsed, tt.parsed) {
				t.Errorf("ParseMessage() = %#v, want %#v", parsed, tt.parsed)
			}
		})
	}
}

func TestParseMessageErrors(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want error
	}{
		{
			name: "empty input",
			in:   nil,
			want: mrrp.ErrShortMessage,
		},
		{
			name: "type 2 short",
			in:   []byte{0x02, 0x00},
			want: mrrp.ErrShortMessage,
		},
		{
			name: "unknown type",
			in:   []byte{0x99},
			want: mrrp.ErrUnknownMessageType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mrrp.ParseMessage(tt.in)
			if err != tt.want {
				t.Errorf("ParseMessage() error = %v, want %v", err, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./web/draft-meow-mrrp-00/ -run "TestMessage|TestParseMessage" -v`
Expected: FAIL — `MeowMeowMeow` etc. undefined.

- [ ] **Step 3: Implement `messages.go`**

```go
package mrrp

import "encoding/binary"

// Message type bytes from the draft.
const (
	TypeMeowMeowMeow         uint8 = 0
	TypeMeowMrrp             uint8 = 1
	TypeMrowMeowMeowMeowMeow uint8 = 2
)

// MeowMeowMeow is the type 0 control message — a single byte with
// value 0.
type MeowMeowMeow struct{}

// Marshal returns the 1-byte wire encoding of m.
func (MeowMeowMeow) Marshal() []byte { return []byte{TypeMeowMeowMeow} }

// Unmarshal parses b into m. b must contain exactly the message type
// byte (and may have trailing bytes, which are ignored).
func (m *MeowMeowMeow) Unmarshal(b []byte) error {
	if len(b) < 1 {
		return ErrShortMessage
	}
	if b[0] != TypeMeowMeowMeow {
		return ErrUnknownMessageType
	}
	*m = MeowMeowMeow{}
	return nil
}

// MeowMrrpMessage is the type 1 control message — a single byte with
// value 1.
type MeowMrrpMessage struct{}

// Marshal returns the 1-byte wire encoding of m.
func (MeowMrrpMessage) Marshal() []byte { return []byte{TypeMeowMrrp} }

// Unmarshal parses b into m.
func (m *MeowMrrpMessage) Unmarshal(b []byte) error {
	if len(b) < 1 {
		return ErrShortMessage
	}
	if b[0] != TypeMeowMrrp {
		return ErrUnknownMessageType
	}
	*m = MeowMrrpMessage{}
	return nil
}

// MrowMeowMeowMeowMeow is the type 2 control message (4 bytes total).
type MrowMeowMeowMeowMeow struct {
	Meeoww uint8  // 1 byte
	MMN    uint16 // Mrrrrow Mrrrrrp Nyaa, 2 bytes, big-endian
}

// Marshal returns the 4-byte wire encoding of m.
func (m MrowMeowMeowMeowMeow) Marshal() []byte {
	b := make([]byte, 4)
	b[0] = TypeMrowMeowMeowMeowMeow
	b[1] = m.Meeoww
	binary.BigEndian.PutUint16(b[2:4], m.MMN)
	return b
}

// Unmarshal parses the 4-byte wire encoding into m.
func (m *MrowMeowMeowMeowMeow) Unmarshal(b []byte) error {
	if len(b) < 4 {
		return ErrShortMessage
	}
	if b[0] != TypeMrowMeowMeowMeowMeow {
		return ErrUnknownMessageType
	}
	m.Meeoww = b[1]
	m.MMN = binary.BigEndian.Uint16(b[2:4])
	return nil
}

// ParseMessage dispatches on b[0] and returns a typed control
// message value (MeowMeowMeow, MeowMrrpMessage, or
// MrowMeowMeowMeowMeow). It returns ErrShortMessage if b is too
// short for the indicated type and ErrUnknownMessageType if b[0] is
// not a known type byte.
func ParseMessage(b []byte) (any, error) {
	if len(b) < 1 {
		return nil, ErrShortMessage
	}
	switch b[0] {
	case TypeMeowMeowMeow:
		var m MeowMeowMeow
		if err := m.Unmarshal(b); err != nil {
			return nil, err
		}
		return m, nil
	case TypeMeowMrrp:
		var m MeowMrrpMessage
		if err := m.Unmarshal(b); err != nil {
			return nil, err
		}
		return m, nil
	case TypeMrowMeowMeowMeowMeow:
		var m MrowMeowMeowMeowMeow
		if err := m.Unmarshal(b); err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, ErrUnknownMessageType
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./web/draft-meow-mrrp-00/ -v`
Expected: PASS — all tests in the package, including the new message tests.

- [ ] **Step 5: Commit**

```bash
git add web/draft-meow-mrrp-00/messages.go web/draft-meow-mrrp-00/messages_test.go
git commit --signoff -m "$(cat <<'EOF'
feat(mrrp): add control messages (types 0, 1, 2) and ParseMessage

Implements the three ICMP-shaped control messages from the draft —
MeowMeowMeow (type 0), MeowMrrpMessage (type 1), and
MrowMeowMeowMeowMeow (type 2) — plus a ParseMessage dispatcher that
returns the typed value or ErrUnknownMessageType.

Assisted-by: Claude Opus 4.7 via Claude Code
Reviewbot-request: yes
EOF
)"
```

---

## Task 7: Final verification

**Files:** none modified.

- [ ] **Step 1: Run full repo test suite**

Run: `npm test`
Expected: PASS — `go generate` produces no diff, `go test ./...` passes including the new package.

- [ ] **Step 2: Run formatter**

Run: `npm run format`
Expected: clean exit, no diff against the just-committed state. If goimports/prettier rewrites anything, commit it as `chore: format`.

- [ ] **Step 3: Confirm the package passes `go vet`**

Run: `go vet ./web/draft-meow-mrrp-00/...`
Expected: clean exit.

---

## Self-Review Notes

**Spec coverage:**

- Package layout (spec §"Package layout"): Tasks 1, 3, 4, 6 create all listed files except the per-package LICENSE, which the plan intentionally drops (sibling pattern: most `web/*` packages have no per-package LICENSE).
- Public API (spec §"Public API"): `MeowHeader` Task 4+5; `MEowPseudoHeader` Task 3; flag constants Task 1; control messages and `ParseMessage` Task 6; `Checksum` Task 2; errors Task 1.
- Wire-format details (spec §"Wire-format details"): exercised by Marshal byte-level assertions in Task 4 (which check exact offsets and big-endian encoding) and pseudo-header tests in Task 3.
- Testing (spec §"Testing"): round-trip Task 5; checksum golden values + corruption Task 2 + Task 5; error cases Task 5 + Task 6; flag encoding covered by the "minimal" and "with options" cases in Task 4 (single flag and multi-flag combinations).

**Type consistency:** `MeowHeader`, `MEowPseudoHeader`, `MeowFlag`, `Checksum` and the message types use the same names everywhere they appear. Field names match the spec character-for-character.

**Placeholder scan:** No "TBD"/"TODO" in the plan. Every code step contains the actual code.
