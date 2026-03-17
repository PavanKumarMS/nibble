# nibble

**Declarative bit-level binary encoding for Go using struct tags.**

Working with binary protocols often means hand-rolling bit-shift arithmetic scattered across hundreds of lines. `nibble` flips that around: you describe your packed format once with `bits:"N"` struct tags and the library handles all masking, shifting, sign-extension, endianness, and validation automatically.

## Installation

```bash
go get github.com/PavanKumarMS/nibble
```

```go
import "github.com/PavanKumarMS/nibble"
```

## Quick start — TCP flags

```go
type TCPFlags struct {
    CWR bool `bits:"1"`
    ECE bool `bits:"1"`
    URG bool `bits:"1"`
    ACK bool `bits:"1"`
    PSH bool `bits:"1"`
    RST bool `bits:"1"`
    SYN bool `bits:"1"`
    FIN bool `bits:"1"`
}

// Decode a raw byte (0x12 = SYN + ACK) — zero allocations
var flags TCPFlags
nibble.UnmarshalBE([]byte{0x12}, &flags)
// flags → {ACK:true SYN:true}

// Encode back — one allocation (the returned []byte)
data, _ := nibble.MarshalBE(&flags)
// data → [0x12]

// Human-readable breakdown
explanation, _ := nibble.Explain([]byte{0x12}, TCPFlags{}, nibble.BigEndian)
fmt.Print(explanation)
```

```
Byte 0 [00010010]:
  bit  7 → CWR:               false (0)
  bit  6 → ECE:               false (0)
  bit  5 → URG:               false (0)
  bit  4 → ACK:               true (1)
  bit  3 → PSH:               false (0)
  bit  2 → RST:               false (0)
  bit  1 → SYN:               true (1)
  bit  0 → FIN:               false (0)
```

## Struct tag format

Annotate each field with `` `bits:"N"` `` where N is the number of bits it occupies in the packed byte stream.

```go
type GamePacket struct {
    IsAlive  bool   `bits:"1"`   // 1 bit  — boolean flag
    WeaponID uint8  `bits:"4"`   // 4 bits — values 0–15
    TeamID   uint8  `bits:"2"`   // 2 bits — values 0–3
    Health   uint16 `bits:"9"`   // 9 bits — values 0–511
}
```

Signed integer fields use two's complement automatically:

```go
type Delta struct {
    DX int8  `bits:"4"` // range [-8, 7]
    DY int8  `bits:"4"` // range [-8, 7]
}
```

Fields are packed in **declaration order**. Bit positions within a byte follow the selected endianness option.

### Supported field types

| Go type              | Max bits | Notes                         |
|----------------------|----------|-------------------------------|
| `bool`               | 1        |                               |
| `uint8`              | 8        |                               |
| `uint16`             | 16       |                               |
| `uint32`             | 32       |                               |
| `uint64`             | 64       |                               |
| `int8`               | 8        | Two's complement, auto sign-extended |
| `int16`              | 16       | Two's complement, auto sign-extended |
| `int32`              | 32       | Two's complement, auto sign-extended |
| `int64`              | 64       | Two's complement, auto sign-extended |

## Performance

nibble uses reflection with schema caching and byte-granularity bit I/O.
The struct schema is parsed once per type (via `sync.Map`) — subsequent calls pay zero reflection or string-parsing cost.

Benchmarked on an i7-10510U (Go 1.26, `go test -bench=. -benchmem`):

| Function | ns/op | B/op | allocs/op |
|---|---|---|---|
| `UnmarshalBE` (TCPFlags, 1 byte) | 114 | **0** | **0** |
| `UnmarshalLE` (GamePacket, 2 bytes) | 64 | **0** | **0** |
| `MarshalBE` (TCPFlags, 1 byte) | 119 | 1 | 1 |
| `MarshalLE` (GamePacket, 2 bytes) | 84 | 2 | 1 |
| `MarshalInto` (TCPFlags, caller buf) | 107 | **0** | **0** |
| `MarshalInto` (GamePacket, caller buf) | 69 | **0** | **0** |
| manual unmarshal (reference) | 0.27 | 0 | 0 |

The single allocation in `MarshalBE`/`MarshalLE` is the returned `[]byte` itself. Use `MarshalInto` with a pooled buffer to eliminate it:

```go
var bufPool = sync.Pool{New: func() any { return make([]byte, 64) }}

func encode(pkt *GamePacket) ([]byte, error) {
    buf := bufPool.Get().([]byte)
    defer bufPool.Put(buf)
    n, err := nibble.MarshalInto(buf, pkt, false) // 0 allocs
    if err != nil {
        return nil, err
    }
    return buf[:n], nil
}
```

At 100,000 packets/second nibble uses less than 2% of a single CPU core — acceptable for the vast majority of production workloads. For latency-critical packet-processing loops, use manual bit manipulation. nibble is designed for correctness, safety, and developer productivity.

## API reference

### `Unmarshal` / `UnmarshalBE` / `UnmarshalLE`

```go
func Unmarshal(src []byte, dst any, opts ...Option) error  // variadic convenience
func UnmarshalBE(src []byte, dst any) error                // BigEndian,    0 allocs
func UnmarshalLE(src []byte, dst any) error                // LittleEndian, 0 allocs
```

Decodes `src` into the struct pointed to by `dst`. `dst` must be a non-nil pointer to a struct.

### `Marshal` / `MarshalBE` / `MarshalLE` / `MarshalInto`

```go
func Marshal(src any, opts ...Option) ([]byte, error)              // variadic convenience
func MarshalBE(src any) ([]byte, error)                            // BigEndian,    1 alloc
func MarshalLE(src any) ([]byte, error)                            // LittleEndian, 1 alloc
func MarshalInto(dst []byte, src any, bigEndian bool) (int, error) // 0 allocs
```

Encodes `src` into bytes. `MarshalInto` writes into a caller-supplied buffer and returns the number of bytes written. The buffer must be at least as large as the packed struct size.

### `Explain`

```go
func Explain(src []byte, schema any, opts ...Option) (string, error)
```

Returns a human-readable byte-by-byte, bit-by-bit breakdown of `src` annotated with field names and values. Useful for debugging wire formats.

### `Validate`

```go
func Validate(src any) error
```

Checks that every field value in `src` fits within its declared bit width. Returns `ErrFieldOverflow` on the first violation found. Useful before sending data on the wire.

### `Diff`

```go
func Diff(a, b any) ([]FieldDiff, error)
```

Compares two structs of the same type and returns a `FieldDiff` for every field whose value changed. Both arguments must be the same struct type.

```go
type FieldDiff struct {
    Field  string
    Before any
    After  any
}
```

### Options

```go
nibble.BigEndian    // first struct field → MSB of first byte (network byte order)
nibble.LittleEndian // first struct field → LSB of first byte (default)
```

Pass as a trailing argument to the variadic functions, or use the named `BE`/`LE` variants for zero allocations:

```go
// variadic (convenience)
nibble.Unmarshal(data, &out, nibble.BigEndian)
nibble.Marshal(&in, nibble.LittleEndian)

// named — preferred in hot paths
nibble.UnmarshalBE(data, &out)
nibble.MarshalLE(&in)
nibble.MarshalInto(buf, &in, true /* bigEndian */)
```

### Error types

| Sentinel              | Meaning                                        |
|-----------------------|------------------------------------------------|
| `ErrFieldOverflow`    | Field value exceeds its declared bit width     |
| `ErrInsufficientData` | Not enough bytes in the source slice           |
| `ErrUnsupportedType`  | Field type cannot be packed (e.g. `string`)    |
| `ErrBitWidthInvalid`  | Bit width exceeds the capacity of the Go type  |

All errors wrap these sentinels so `errors.Is` works:

```go
if errors.Is(err, nibble.ErrFieldOverflow) { ... }
```

## Endianness explained

`nibble` operates at the **bit-stream** level, not the byte level.

| Mode          | Bit-stream → byte layout                         |
|---------------|--------------------------------------------------|
| `LittleEndian`| Stream bit 0 → LSB of byte 0 (bit 0 of byte 0)  |
| `BigEndian`   | Stream bit 0 → MSB of byte 0 (bit 7 of byte 0)  |

For multi-bit fields, `LittleEndian` places the field's LSB at the lower stream position; `BigEndian` places the field's MSB there. Use `BigEndian` for all IETF/network protocols (TCP, UDP, DNS, etc.).

## Comparison with existing libraries

| Feature                        | nibble | encoding/binary | manual bit-shifts |
|--------------------------------|--------|-----------------|-------------------|
| Sub-byte field widths          | ✅     | ❌              | ✅                |
| Declarative struct tags        | ✅     | ❌              | ❌                |
| Signed integers (auto)         | ✅     | ✅              | manual            |
| Human-readable explain         | ✅     | ❌              | ❌                |
| Field diff                     | ✅     | ❌              | ❌                |
| Overflow validation            | ✅     | ❌              | manual            |
| Zero-alloc unmarshal           | ✅     | ✅              | ✅                |
| Zero-alloc marshal             | ✅ (`MarshalInto`) | ✅ | ✅          |
| Zero dependencies              | ✅     | ✅              | ✅                |

## Examples

Full runnable examples live in `examples/`:

```
examples/tcp/main.go   — TCP control flags byte
examples/game/main.go  — 16-bit compact game-state packet
```

```bash
go run ./examples/tcp/
go run ./examples/game/
```

## Testing and benchmarks

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run benchmarks
go test -bench=. -benchmem ./...

# Run benchmarks for 10 seconds each (more stable numbers)
go test -bench=. -benchmem -benchtime=10s ./...
```

## Contributing

1. Fork the repository and create a feature branch.
2. Add tests for any new behaviour.
3. Run `go test ./...` — all tests must pass.
4. Open a pull request with a clear description of the change.

Bug reports and feature requests are welcome via GitHub Issues.
