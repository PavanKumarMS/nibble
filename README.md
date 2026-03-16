# bitpack

**Declarative bit-level binary encoding for Go using struct tags.**

Working with binary protocols often means hand-rolling bit-shift arithmetic scattered across hundreds of lines.  `bitpack` flips that around: you describe your packed format once with `bits:"N"` struct tags and the library handles all masking, shifting, sign-extension, endianness, and validation automatically.

## Installation

```bash
go get github.com/pavankumarms/nibble
```

Import as:

```go
import bitpack "github.com/pavankumarms/nibble"
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

// Decode a raw byte (0x12 = SYN + ACK)
var flags TCPFlags
bitpack.Unmarshal([]byte{0x12}, &flags, bitpack.BigEndian)
// flags → {ACK:true SYN:true}

// Encode back
data, _ := bitpack.Marshal(&flags, bitpack.BigEndian)
// data → [0x12]

// Human-readable breakdown
explanation, _ := bitpack.Explain([]byte{0x12}, TCPFlags{}, bitpack.BigEndian)
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

```go
type GamePacket struct {
    IsAlive  bool   `bits:"1"`   // 1 bit  — boolean flag
    WeaponID uint8  `bits:"4"`   // 4 bits — values 0-15
    TeamID   uint8  `bits:"2"`   // 2 bits — values 0-3
    Health   uint16 `bits:"9"`   // 9 bits — values 0-511
}
```

Fields are packed in **declaration order**.  Bit positions within a byte follow the selected endianness option.

### Supported field types

| Go type              | Max bits |
|----------------------|----------|
| `bool`               | 1        |
| `uint8`              | 8        |
| `uint16`             | 16       |
| `uint32`             | 32       |
| `uint64`             | 64       |
| `int8`               | 8        |
| `int16`              | 16       |
| `int32`              | 32       |
| `int64`              | 64       |

Signed integers use two's complement encoding automatically.

## API reference

### `Unmarshal`

```go
func Unmarshal(src []byte, dst any, opts ...Option) error
```

Decodes `src` into the struct pointed to by `dst`.

### `Marshal`

```go
func Marshal(src any, opts ...Option) ([]byte, error)
```

Encodes the struct `src` into a minimal byte slice.

### `Explain`

```go
func Explain(src []byte, schema any, opts ...Option) (string, error)
```

Returns a human-readable annotation of which bytes and bits correspond to which struct fields.

### `Validate`

```go
func Validate(src any) error
```

Checks that every field value fits within its declared bit width.  Returns `ErrFieldOverflow` on the first violation found.

### `Diff`

```go
func Diff(a, b any) ([]FieldDiff, error)
```

Compares two structs of the same type and returns a `FieldDiff` for every field whose value changed.

```go
type FieldDiff struct {
    Field  string
    Before any
    After  any
}
```

### Options

```go
bitpack.BigEndian    // first field → MSB (network byte order)
bitpack.LittleEndian // first field → LSB (default)
```

Pass as trailing variadic arguments:

```go
bitpack.Unmarshal(data, &out, bitpack.BigEndian)
bitpack.Marshal(&in, bitpack.LittleEndian)
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
if errors.Is(err, bitpack.ErrFieldOverflow) { ... }
```

## Endianness explained

`bitpack` operates at the **bit-stream** level, not the byte level.

| Mode          | Bit-stream → byte layout                        |
|---------------|--------------------------------------------------|
| `LittleEndian`| Stream bit 0 → LSB of byte 0 (bit 0 of byte 0) |
| `BigEndian`   | Stream bit 0 → MSB of byte 0 (bit 7 of byte 0) |

For multi-bit fields, `LittleEndian` places the field's LSB at the lower stream position; `BigEndian` places the field's MSB there.  Use `BigEndian` for all IETF/network protocols.

## Comparison with existing libraries

| Feature                        | bitpack | encoding/binary | manual bit-shifts |
|--------------------------------|---------|-----------------|-------------------|
| Sub-byte field widths          | ✅      | ❌              | ✅                |
| Declarative struct tags        | ✅      | ❌              | ❌                |
| Signed integers                | ✅      | ✅              | manual            |
| Human-readable explain         | ✅      | ❌              | ❌                |
| Field diff                     | ✅      | ❌              | ❌                |
| Overflow validation            | ✅      | ❌              | manual            |
| Zero dependencies              | ✅      | ✅              | ✅                |

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

## Contributing

1. Fork the repository and create a feature branch.
2. Add tests for any new behaviour.
3. Run `go test ./...` — all tests must pass.
4. Open a pull request with a clear description of the change.

Bug reports and feature requests are welcome via GitHub Issues.
