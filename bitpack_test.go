package bitpack_test

import (
	"errors"
	"strings"
	"testing"

	bitpack "github.com/pavankumarms/nibble"
)

// ── Shared test types ──────────────────────────────────────────────────────

// TCPFlags matches the TCP control bits byte (network / big-endian order:
// CWR is the most-significant bit, FIN is the least-significant).
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

// GamePacket has mixed field sizes spanning more than one byte.
type GamePacket struct {
	IsAlive  bool   `bits:"1"`
	WeaponID uint8  `bits:"4"`
	TeamID   uint8  `bits:"2"`
	Health   uint16 `bits:"9"`
}

// SignedPacket exercises signed integer fields.
type SignedPacket struct {
	A int8  `bits:"4"` // range [-8, 7]
	B int16 `bits:"4"` // range [-8, 7]
}

// ── 1. TCP flags from known hex bytes ─────────────────────────────────────

// TCP flags byte 0x12 = 0b00010010 in network (big-endian) byte order means:
//   CWR=0 ECE=0 URG=0 ACK=1 PSH=0 RST=0 SYN=1 FIN=0
func TestUnmarshalTCPFlags(t *testing.T) {
	data := []byte{0x12} // 0b00010010

	var flags TCPFlags
	if err := bitpack.Unmarshal(data, &flags, bitpack.BigEndian); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if flags.ACK != true {
		t.Errorf("ACK: want true, got false")
	}
	if flags.SYN != true {
		t.Errorf("SYN: want true, got false")
	}
	if flags.FIN || flags.RST || flags.PSH || flags.URG || flags.ECE || flags.CWR {
		t.Errorf("unexpected flags set: %+v", flags)
	}
}

// ── 2. Game packet with mixed field sizes ──────────────────────────────────

func TestUnmarshalGamePacket(t *testing.T) {
	// Manually construct a 2-byte (16-bit) game packet in little-endian order.
	// Fields (LSB-first within each byte):
	//   IsAlive  = 1   (bit 0)
	//   WeaponID = 5   (bits 1-4)  → 0101
	//   TeamID   = 2   (bits 5-6)  → 10
	//   Health   = 300 (bits 7-15) → 100101100
	//
	// Bit stream (bit 0 = LSB of byte 0):
	//   pos  0 : IsAlive=1
	//   pos  1-4: WeaponID=5  (0101 LSB-first → 1,0,1,0)
	//   pos  5-6: TeamID=2    (10   LSB-first → 0,1)
	//   pos  7-15: Health=300 (100101100 LSB-first → 0,0,1,1,0,1,0,0,1)
	//
	// byte 0 bits (0..7): 1, 1,0,1,0, 0,1, 0  → 0b01001011 = 0x4B
	// byte 1 bits (8..15): 0,0,1,1,0,1,0,0    → wait, let me recalculate.
	//
	// bit stream positional (0=LSB byte0 … 15=MSB byte1):
	//   [0]=1  [1]=1 [2]=0 [3]=1 [4]=0  [5]=0 [6]=1  [7]=0 [8]=0 [9]=1 [10]=1 [11]=0 [12]=1 [13]=0 [14]=0 [15]=1
	//   IsAlive WeaponID=5(1010->read as lsb first 1+0*2+1*4+0*8=5) TeamID=2(01->0+1*2=2) Health=300
	//
	// byte0 = bit7..bit0 = 0,1,0,1,0,1,1 → wait I need to be careful.
	// byte0[bit0]=1 byte0[bit1]=1 byte0[bit2]=0 byte0[bit3]=1 byte0[bit4]=0 byte0[bit5]=0 byte0[bit6]=1 byte0[bit7]=0
	// byte0 = bit7*128 + ... + bit0*1 = 0*128+1*64+0*32+0*16+1*8+0*4+1*2+1*1 = 64+8+2+1 = 75 = 0x4B
	//
	// Health=300 = 0b100101100, LSB-first in positions 7..15:
	//   pos7 = 0 (300 bit0), pos8 = 0 (300 bit1), pos9 = 1 (300 bit2), pos10 = 1 (300 bit3),
	//   pos11 = 0 (300 bit4), pos12 = 1 (300 bit5), pos13 = 0 (300 bit6), pos14 = 0 (300 bit7), pos15 = 1 (300 bit8)
	// byte1[bit0]=pos8=0 byte1[bit1]=pos9=1 byte1[bit2]=pos10=1 byte1[bit3]=pos11=0
	//       byte1[bit4]=pos12=1 byte1[bit5]=pos13=0 byte1[bit6]=pos14=0 byte1[bit7]=pos15=1
	// byte1 = 1*128+0*64+0*32+1*16+0*8+1*4+1*2+0*1 = 128+16+4+2 = 150 = 0x96
	//
	// pos7 belongs to byte0[bit7]=0 (Health bit0=0)
	// byte0 = 64+8+2+1 = 0x4B  (unchanged, bit7=0)
	data := []byte{0x4B, 0x96}

	var pkt GamePacket
	if err := bitpack.Unmarshal(data, &pkt, bitpack.LittleEndian); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !pkt.IsAlive {
		t.Errorf("IsAlive: want true")
	}
	if pkt.WeaponID != 5 {
		t.Errorf("WeaponID: want 5, got %d", pkt.WeaponID)
	}
	if pkt.TeamID != 2 {
		t.Errorf("TeamID: want 2, got %d", pkt.TeamID)
	}
	if pkt.Health != 300 {
		t.Errorf("Health: want 300, got %d", pkt.Health)
	}
}

// ── 3. Round-trip: Marshal → Unmarshal ────────────────────────────────────

func TestRoundTripTCPFlags(t *testing.T) {
	original := TCPFlags{ACK: true, SYN: true, FIN: true}

	data, err := bitpack.Marshal(&original, bitpack.BigEndian)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded TCPFlags
	if err := bitpack.Unmarshal(data, &decoded, bitpack.BigEndian); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripGamePacket(t *testing.T) {
	original := GamePacket{IsAlive: true, WeaponID: 13, TeamID: 3, Health: 511}

	data, err := bitpack.Marshal(&original, bitpack.LittleEndian)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded GamePacket
	if err := bitpack.Unmarshal(data, &decoded, bitpack.LittleEndian); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripSigned(t *testing.T) {
	original := SignedPacket{A: -3, B: 7}

	data, err := bitpack.Marshal(&original, bitpack.LittleEndian)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded SignedPacket
	if err := bitpack.Unmarshal(data, &decoded, bitpack.LittleEndian); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// ── 4. Explain output ──────────────────────────────────────────────────────

func TestExplain(t *testing.T) {
	data := []byte{0x12} // SYN + ACK set, big-endian TCP byte

	out, err := bitpack.Explain(data, TCPFlags{}, bitpack.BigEndian)
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}

	// Must contain byte header and all field names.
	for _, want := range []string{"Byte 0", "CWR", "ECE", "URG", "ACK", "PSH", "RST", "SYN", "FIN"} {
		if !strings.Contains(out, want) {
			t.Errorf("Explain output missing %q\nFull output:\n%s", want, out)
		}
	}

	// ACK and SYN should show true, others false.
	// The output format is: `bit N → Field: true (1)` or `false (0)`.
	if !strings.Contains(out, "SYN") {
		t.Error("missing SYN in explain output")
	}
}

func TestExplainMultiByte(t *testing.T) {
	pkt := GamePacket{IsAlive: true, WeaponID: 5, TeamID: 2, Health: 300}
	data, err := bitpack.Marshal(&pkt, bitpack.LittleEndian)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	out, err := bitpack.Explain(data, GamePacket{}, bitpack.LittleEndian)
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}

	for _, want := range []string{"Byte 0", "Byte 1", "IsAlive", "WeaponID", "TeamID", "Health"} {
		if !strings.Contains(out, want) {
			t.Errorf("Explain output missing %q\nFull output:\n%s", want, out)
		}
	}
}

// ── 5. Validate catches overflow ───────────────────────────────────────────

func TestValidateOverflow(t *testing.T) {
	// WeaponID is 4 bits → max 15.
	pkt := GamePacket{WeaponID: 16}
	err := bitpack.Validate(&pkt)
	if err == nil {
		t.Fatal("expected ErrFieldOverflow, got nil")
	}
	if !errors.Is(err, bitpack.ErrFieldOverflow) {
		t.Errorf("expected ErrFieldOverflow, got %v", err)
	}
}

func TestValidateOK(t *testing.T) {
	pkt := GamePacket{IsAlive: true, WeaponID: 15, TeamID: 3, Health: 511}
	if err := bitpack.Validate(&pkt); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSignedOverflow(t *testing.T) {
	// A is int8 with 4 bits → range [-8, 7]; value 8 overflows.
	pkt := SignedPacket{A: 8}
	err := bitpack.Validate(&pkt)
	if err == nil {
		t.Fatal("expected ErrFieldOverflow, got nil")
	}
	if !errors.Is(err, bitpack.ErrFieldOverflow) {
		t.Errorf("expected ErrFieldOverflow, got %v", err)
	}
}

// ── 6. Diff detects changed fields ────────────────────────────────────────

func TestDiffChanged(t *testing.T) {
	a := TCPFlags{SYN: true, ACK: false}
	b := TCPFlags{SYN: true, ACK: true}

	diffs, err := bitpack.Diff(a, b)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("want 1 diff, got %d: %+v", len(diffs), diffs)
	}
	if diffs[0].Field != "ACK" {
		t.Errorf("want diff on ACK, got %q", diffs[0].Field)
	}
	if diffs[0].Before != false {
		t.Errorf("Before: want false, got %v", diffs[0].Before)
	}
	if diffs[0].After != true {
		t.Errorf("After: want true, got %v", diffs[0].After)
	}
}

func TestDiffNoDiff(t *testing.T) {
	a := TCPFlags{SYN: true}
	b := TCPFlags{SYN: true}

	diffs, err := bitpack.Diff(a, b)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected no diffs, got %+v", diffs)
	}
}

func TestDiffMultipleFields(t *testing.T) {
	a := GamePacket{IsAlive: true, WeaponID: 3, TeamID: 1, Health: 100}
	b := GamePacket{IsAlive: false, WeaponID: 7, TeamID: 1, Health: 200}

	diffs, err := bitpack.Diff(a, b)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	// IsAlive, WeaponID, Health changed; TeamID did not.
	if len(diffs) != 3 {
		t.Fatalf("want 3 diffs, got %d: %+v", len(diffs), diffs)
	}
}

// ── 7. BigEndian vs LittleEndian produce different bytes ───────────────────

func TestEndiannessProducesDifferentBytes(t *testing.T) {
	pkt := GamePacket{IsAlive: true, WeaponID: 5, TeamID: 2, Health: 300}

	le, err := bitpack.Marshal(&pkt, bitpack.LittleEndian)
	if err != nil {
		t.Fatalf("Marshal LE: %v", err)
	}
	be, err := bitpack.Marshal(&pkt, bitpack.BigEndian)
	if err != nil {
		t.Fatalf("Marshal BE: %v", err)
	}

	same := true
	for i := range le {
		if le[i] != be[i] {
			same = false
			break
		}
	}
	if same {
		t.Errorf("BigEndian and LittleEndian produced identical bytes: %x", le)
	}

	// Each encoding must round-trip in its own mode.
	var leOut, beOut GamePacket
	if err := bitpack.Unmarshal(le, &leOut, bitpack.LittleEndian); err != nil {
		t.Fatalf("Unmarshal LE: %v", err)
	}
	if err := bitpack.Unmarshal(be, &beOut, bitpack.BigEndian); err != nil {
		t.Fatalf("Unmarshal BE: %v", err)
	}
	if leOut != pkt {
		t.Errorf("LE round-trip: got %+v, want %+v", leOut, pkt)
	}
	if beOut != pkt {
		t.Errorf("BE round-trip: got %+v, want %+v", beOut, pkt)
	}
}

// ── 8. Error cases ─────────────────────────────────────────────────────────

func TestErrInsufficientData(t *testing.T) {
	err := bitpack.Unmarshal([]byte{}, &TCPFlags{}, bitpack.BigEndian)
	if !errors.Is(err, bitpack.ErrInsufficientData) {
		t.Errorf("expected ErrInsufficientData, got %v", err)
	}
}

func TestErrBitWidthInvalid(t *testing.T) {
	type Bad struct {
		X uint8 `bits:"9"` // uint8 can only hold 8 bits
	}
	err := bitpack.Unmarshal([]byte{0xFF, 0xFF}, &Bad{})
	if !errors.Is(err, bitpack.ErrBitWidthInvalid) {
		t.Errorf("expected ErrBitWidthInvalid, got %v", err)
	}
}

func TestErrUnsupportedType(t *testing.T) {
	type Bad struct {
		X string `bits:"8"`
	}
	err := bitpack.Unmarshal([]byte{0xFF}, &Bad{})
	if !errors.Is(err, bitpack.ErrUnsupportedType) {
		t.Errorf("expected ErrUnsupportedType, got %v", err)
	}
}

func TestErrMarshalOverflow(t *testing.T) {
	// TeamID is 2 bits → max 3; value 4 must fail.
	pkt := GamePacket{TeamID: 4}
	_, err := bitpack.Marshal(&pkt)
	if !errors.Is(err, bitpack.ErrFieldOverflow) {
		t.Errorf("expected ErrFieldOverflow, got %v", err)
	}
}

func TestDiffTypeMismatch(t *testing.T) {
	_, err := bitpack.Diff(TCPFlags{}, GamePacket{})
	if err == nil {
		t.Fatal("expected error for type mismatch, got nil")
	}
}

// ── Byte-level correctness for a known TCP SYN-ACK byte ───────────────────

// 0x12 = 0b00010010 big-endian:
//
//	bit7(CWR)=0 bit6(ECE)=0 bit5(URG)=0 bit4(ACK)=1 bit3(PSH)=0 bit2(RST)=0 bit1(SYN)=1 bit0(FIN)=0
func TestMarshalTCPSYNACK(t *testing.T) {
	flags := TCPFlags{ACK: true, SYN: true}
	data, err := bitpack.Marshal(&flags, bitpack.BigEndian)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 byte, got %d", len(data))
	}
	if data[0] != 0x12 {
		t.Errorf("expected 0x12, got 0x%02x (%08b)", data[0], data[0])
	}
}
