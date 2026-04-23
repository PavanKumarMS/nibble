package nibble_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/PavanKumarMS/nibble"
)

// ── Skip/padding fields ────────────────────────────────────────────────────

// bits:"-" excludes the field from packing entirely.
type SkipField struct {
	A       uint8  `bits:"4"`
	Ignored string `bits:"-"` // excluded: no bits consumed
	B       uint8  `bits:"4"`
}

func TestSkipField_RoundTrip(t *testing.T) {
	original := SkipField{A: 5, B: 3}
	data, err := nibble.MarshalLE(&original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 byte (8 bits), got %d", len(data))
	}
	var out SkipField
	if err := nibble.UnmarshalLE(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.A != original.A || out.B != original.B {
		t.Errorf("round-trip: got %+v, want %+v", out, original)
	}
}

// Blank identifier padding field: _ T `bits:"N"` consumes bits without a name.
type PaddedHeader struct {
	Version  uint8 `bits:"4"`
	_        uint8 `bits:"4"` // reserved — consume 4 bits, value ignored
	Checksum uint8 `bits:"8"`
}

func TestPaddingField_RoundTrip(t *testing.T) {
	original := PaddedHeader{Version: 7, Checksum: 0xAB}
	data, err := nibble.MarshalBE(&original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 bytes (16 bits), got %d", len(data))
	}
	var out PaddedHeader
	if err := nibble.UnmarshalBE(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Version != original.Version || out.Checksum != original.Checksum {
		t.Errorf("round-trip: got %+v, want %+v", out, original)
	}
}

func TestPaddingField_PaddedBitsAreZero(t *testing.T) {
	pkt := PaddedHeader{Version: 3, Checksum: 0xFF}
	data, err := nibble.MarshalBE(&pkt)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// BigEndian byte 0: bits 7-4 = Version=3 (0011), bits 3-0 = pad = 0 (0000) → 0b00110000 = 0x30
	if data[0] != 0x30 {
		t.Errorf("byte 0: want 0x30, got 0x%02x", data[0])
	}
	// byte 1: Checksum = 0xFF
	if data[1] != 0xFF {
		t.Errorf("byte 1: want 0xFF, got 0x%02x", data[1])
	}
}

// ── Array fields ───────────────────────────────────────────────────────────

type ColorPalette struct {
	Pixels [4]uint8 `bits:"4"` // four 4-bit colour indices
}

func TestArrayField_RoundTrip(t *testing.T) {
	original := ColorPalette{Pixels: [4]uint8{1, 5, 9, 15}}
	data, err := nibble.MarshalLE(&original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 bytes (4×4 bits), got %d", len(data))
	}
	var out ColorPalette
	if err := nibble.UnmarshalLE(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out != original {
		t.Errorf("round-trip: got %+v, want %+v", out, original)
	}
}

func TestArrayField_Overflow(t *testing.T) {
	pkt := ColorPalette{Pixels: [4]uint8{0, 0, 0, 16}} // 16 overflows 4 bits
	_, err := nibble.MarshalLE(&pkt)
	if err == nil {
		t.Fatal("expected ErrFieldOverflow, got nil")
	}
}

type FlagArray struct {
	Flags [8]bool `bits:"1"`
}

func TestArrayField_Bools(t *testing.T) {
	original := FlagArray{Flags: [8]bool{true, false, true, false, true, false, true, false}}
	data, err := nibble.MarshalLE(&original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 byte, got %d", len(data))
	}
	var out FlagArray
	if err := nibble.UnmarshalLE(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out != original {
		t.Errorf("round-trip: got %+v, want %+v", out, original)
	}
}

// ── Nested structs ─────────────────────────────────────────────────────────

type EtherType struct {
	SYN bool `bits:"1"`
	ACK bool `bits:"1"`
	FIN bool `bits:"1"`
	RST bool `bits:"1"`
}

type IPPacket struct {
	Version uint8     `bits:"4"`
	Flags   EtherType // nested: 4 bits inlined
}

func TestNestedStruct_RoundTrip(t *testing.T) {
	original := IPPacket{
		Version: 4,
		Flags:   EtherType{SYN: true, ACK: true},
	}
	data, err := nibble.MarshalBE(&original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 byte (4+4 bits), got %d", len(data))
	}
	var out IPPacket
	if err := nibble.UnmarshalBE(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out != original {
		t.Errorf("round-trip: got %+v, want %+v", out, original)
	}
}

func TestNestedStruct_BitLayout(t *testing.T) {
	// BigEndian: Version (4 bits MSB) then Flags (4 bits LSB) in byte 0.
	// Version=6 (0110), SYN=1, ACK=0, FIN=0, RST=0 → 0b01100100 = 0x64... wait
	// BE: bit0=MSB of byte0 → Version MSB first: 0110, then SYN=1,ACK=0,FIN=0,RST=0
	// byte0 = 0110_1000 = 0x68
	pkt := IPPacket{Version: 6, Flags: EtherType{SYN: true}}
	data, err := nibble.MarshalBE(&pkt)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if data[0] != 0x68 {
		t.Errorf("expected 0x68, got 0x%02x (%08b)", data[0], data[0])
	}
}

func TestNestedStruct_Validate(t *testing.T) {
	pkt := IPPacket{Version: 16} // 16 overflows 4 bits
	if err := nibble.Validate(&pkt); err == nil {
		t.Fatal("expected ErrFieldOverflow, got nil")
	}
}

func TestNestedStruct_Diff(t *testing.T) {
	a := IPPacket{Version: 4, Flags: EtherType{SYN: true}}
	b := IPPacket{Version: 4, Flags: EtherType{SYN: false, ACK: true}}
	diffs, err := nibble.Diff(a, b)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(diffs) != 2 {
		t.Fatalf("want 2 diffs (Flags.SYN, Flags.ACK), got %d: %+v", len(diffs), diffs)
	}
}

// ── Streaming encoder/decoder ──────────────────────────────────────────────

func TestStream_WriteRead_BE(t *testing.T) {
	packets := []TCPFlags{
		{SYN: true},
		{ACK: true},
		{SYN: true, ACK: true, FIN: true},
	}

	var buf bytes.Buffer
	w := nibble.NewWriterBE(&buf)
	for i, pkt := range packets {
		if err := w.Write(&pkt); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}

	r := nibble.NewReaderBE(&buf)
	for i, want := range packets {
		var got TCPFlags
		if err := r.Read(&got); err != nil {
			t.Fatalf("Read[%d]: %v", i, err)
		}
		if got != want {
			t.Errorf("[%d] got %+v, want %+v", i, got, want)
		}
	}
}

func TestStream_WriteRead_LE(t *testing.T) {
	packets := []GamePacket{
		{IsAlive: true, WeaponID: 3, TeamID: 1, Health: 100},
		{IsAlive: false, WeaponID: 7, TeamID: 2, Health: 400},
	}

	var buf bytes.Buffer
	w := nibble.NewWriterLE(&buf)
	for _, pkt := range packets {
		if err := w.Write(pkt); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	r := nibble.NewReaderLE(&buf)
	for i, want := range packets {
		var got GamePacket
		if err := r.Read(&got); err != nil {
			t.Fatalf("Read[%d]: %v", i, err)
		}
		if got != want {
			t.Errorf("[%d] got %+v, want %+v", i, got, want)
		}
	}
}

func TestStream_Read_EOF(t *testing.T) {
	r := nibble.NewReaderBE(bytes.NewReader(nil))
	var flags TCPFlags
	err := r.Read(&flags)
	if err != io.ErrUnexpectedEOF && err != io.EOF {
		t.Errorf("expected EOF or ErrUnexpectedEOF, got %v", err)
	}
}

func TestStream_VariadicOptions(t *testing.T) {
	var buf bytes.Buffer
	w := nibble.NewWriter(&buf, nibble.BigEndian)
	flags := TCPFlags{SYN: true}
	if err := w.Write(&flags); err != nil {
		t.Fatalf("Write: %v", err)
	}

	r := nibble.NewReader(&buf, nibble.BigEndian)
	var got TCPFlags
	if err := r.Read(&got); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != flags {
		t.Errorf("got %+v, want %+v", got, flags)
	}
}
