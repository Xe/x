# draft-meow-mrrp-00 — Go codec

**Date:** 2026-04-17
**Author:** Xe Iaso (with Claude)
**Status:** Approved

## Summary

A Go package at `web/draft-meow-mrrp-00` (import alias `mrrp`) that
marshals and unmarshals frames defined by [draft-meow-mrrp-00][draft], an
April 2026 humorous Internet-Draft whose wire format is structurally
identical to TCP plus an IP-style pseudo-header for checksum
computation, and which also defines three ICMP-shaped control messages
(types 0, 1, 2).

[draft]: https://www.ietf.org/archive/id/draft-meow-mrrp-00.html

The package is a pure codec: it parses bytes into typed structs and
serializes typed structs back to bytes. It performs no network I/O.

## Goals

- Round-trip every frame defined by the draft without loss.
- Compute and validate the checksum (`Mrrrowww`) using the pseudo-header,
  matching the TCP/RFC 1071 1's-complement algorithm the wire format
  implies.
- Use the draft's literal cat-themed field names in the public API. Doc
  comments carry the TCP equivalent for each field, so anyone fluent in
  TCP can map fields immediately.
- Fit in next to the other packages under `web/` (single-purpose
  package, table-driven tests, no surprises).

## Non-goals

- No network I/O, no socket layer, no userspace transport.
- No decoder CLI in this iteration.
- No fuzzing harness in v1 (can be added later if it earns its keep).
- No interop testing — there is no other implementation to interop
  with.

## Package layout

```
web/draft-meow-mrrp-00/
  doc.go         // package doc, draft pointer, TCP analogue note
  mrrp.go        // shared constants, MeowFlag bits, error vars
  header.go      // MeowHeader + Marshal/Unmarshal
  pseudo.go      // MEowPseudoHeader + Marshal
  messages.go    // MeowMeowMeow, MeowMrrpMessage, MrowMeowMeowMeowMeow
                 // + ParseMessage dispatcher
  checksum.go    // Checksum(parts ...[]byte) uint16, RFC 1071-style
  *_test.go      // table-driven golden vectors + round-trip tests
  LICENSE        // matches sibling packages
```

Import path: `within.website/x/web/draft-meow-mrrp-00`.
Import alias: `mrrp`.

## Public API

### Main header

```go
type MeowHeader struct {
    MeowMrrp           uint16   // src port equivalent (16 bits)
    MoewMrrp           uint16   // dst port equivalent (16 bits)
    MeeeeowNyaaaa      uint32   // sequence number (32 bits)
    MrrrreeaowwwwMrrp  uint32   // acknowledgment number (32 bits)
    MeowMrrrrp         uint8    // 4 bits — data offset, in 32-bit words; min 5
    Purrr              uint8    // 4 bits — reserved, MUST be zero on send
    Flags              MeowFlag // 8 bits — see flag constants
    Mraoww             uint16   // window (16 bits)
    Mrrrowww           uint16   // checksum (16 bits) — auto on Marshal
    UrgentMeowing      uint16   // urgent pointer (16 bits)
    Meeeowo            []byte   // options, (MeowMrrrrp-5)*4 bytes; 32-bit aligned
    Meow               []byte   // payload
}

func (h *MeowHeader) Marshal(p MEowPseudoHeader) ([]byte, error)
func (h *MeowHeader) Unmarshal(p MEowPseudoHeader, b []byte) error
```

`Marshal` writes `Mrrrowww` itself (zeroes the field, computes the
checksum over pseudo-header + header + payload, writes the result).
Callers do not set `Mrrrowww`.

`Unmarshal` validates the checksum against the supplied pseudo-header
and returns `ErrChecksumMismatch` on failure. It also derives
`Meeeowo` length from `MeowMrrrrp`, returning `ErrBadDataOffset` if
`MeowMrrrrp < 5` and `ErrOptionsLength` if the implied options length
exceeds the input.

### Flags

```go
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
```

Bit ordering follows the draft's left-to-right order in Figure 1
(MEOW is the most significant of the 8-bit flag byte).

### Pseudo-header

```go
type MEowPseudoHeader struct {
    MeeowwNyaaaaa      [4]byte // source address (4 bytes)
    PurrreeowwwMrrrrrp [4]byte // destination address (4 bytes)
    Miau               uint8   // zero byte
    MRRP               uint8   // protocol byte
    MEowMeeoww         uint16  // length: MeowHeader + payload, in bytes
}

func (p MEowPseudoHeader) Marshal() []byte // 12 bytes, big-endian
```

The pseudo-header is never transmitted. It exists only as input to the
checksum, mirroring the TCP/IP relationship.

### Control messages

```go
type MeowMeowMeow struct{}                // 1 byte: 0x00

type MeowMrrpMessage struct{}             // 1 byte: 0x01

type MrowMeowMeowMeowMeow struct {        // 4 bytes
    Meeoww uint8                          // 1 byte
    MMN    uint16                         // 2 bytes (Mrrrrow Mrrrrrp Nyaa)
}

func (m MeowMeowMeow) Marshal() []byte
func (m *MeowMeowMeow) Unmarshal(b []byte) error
// ... same shape for the other two

func ParseMessage(b []byte) (any, error) // dispatches on b[0]
```

`ParseMessage` returns `ErrUnknownMessageType` if the first byte is
not 0, 1, or 2.

### Checksum

```go
func Checksum(parts ...[]byte) uint16
```

1's-complement sum per RFC 1071. Caller passes pseudo-header bytes,
header bytes (with checksum field zeroed), and payload bytes — order
matches the wire as it would be checksummed.

### Errors

```go
var (
    ErrShortHeader        = errors.New("mrrp: short header")
    ErrBadDataOffset      = errors.New("mrrp: bad data offset")
    ErrOptionsLength      = errors.New("mrrp: options length exceeds input")
    ErrChecksumMismatch   = errors.New("mrrp: checksum mismatch")
    ErrUnknownMessageType = errors.New("mrrp: unknown message type")
)
```

## Wire-format details

All multi-byte integers are big-endian (network byte order), matching
the bit layouts in the draft.

Main header layout (20 bytes minimum + options + payload):

| Offset | Size | Field                      |
| -----: | ---: | -------------------------- |
|      0 |    2 | `MeowMrrp`                 |
|      2 |    2 | `MoewMrrp`                 |
|      4 |    4 | `MeeeeowNyaaaa`            |
|      8 |    4 | `MrrrreeaowwwwMrrp`        |
|     12 |    1 | `MeowMrrrrp`<<4 \| `Purrr` |
|     13 |    1 | `Flags`                    |
|     14 |    2 | `Mraoww`                   |
|     16 |    2 | `Mrrrowww`                 |
|     18 |    2 | `UrgentMeowing`            |
|     20 |  var | `Meeeowo` (options)        |
|    var |  var | `Meow` (payload)           |

Pseudo-header layout (12 bytes):

| Offset | Size | Field                |
| -----: | ---: | -------------------- |
|      0 |    4 | `MeeowwNyaaaaa`      |
|      4 |    4 | `PurrreeowwwMrrrrrp` |
|      8 |    1 | `Miau`               |
|      9 |    1 | `MRRP`               |
|     10 |    2 | `MEowMeeoww`         |

Control messages: dispatched on first byte (0, 1, or 2). Type 0 and
type 1 are 1 byte total. Type 2 is 4 bytes total.

## Testing

Per repo convention (`go-table-driven-tests` skill):

- **Round-trip vectors**: hand-crafted byte slices for each message
  type — `Unmarshal` then `Marshal` must reproduce the input byte for
  byte.
- **Checksum**: golden values for known pseudo-header + header + payload
  combinations, plus a corruption case that asserts
  `ErrChecksumMismatch`.
- **Errors**: short input, `MeowMrrrrp < 5`, options-overrun, unknown
  message type — each gets one row.
- **Flag encoding**: a row per single-flag value plus one combined-flag
  row, asserting the bit position matches the draft.

No fuzzing target in v1 — add only if a real bug suggests it.

## Out of scope

- Decoder CLI (`cmd/meowdump`) — easy to add later if useful.
- Userspace transport over UDP.
- IPv6 pseudo-header variant (the draft only shows the 4-byte address
  layout).
- Anything beyond the wire format. The draft's prose is not
  implementable.
