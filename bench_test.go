package nibble_test

import (
	"testing"

	"github.com/PavanKumarMS/nibble"
)

// ── variadic API (1 alloc for []Option backing array) ─────────────────────

func BenchmarkUnmarshal_TCPFlags(b *testing.B) {
	data := []byte{0x12}
	var flags TCPFlags
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = nibble.Unmarshal(data, &flags, nibble.BigEndian)
	}
}

func BenchmarkMarshal_TCPFlags(b *testing.B) {
	flags := TCPFlags{ACK: true, SYN: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = nibble.Marshal(&flags, nibble.BigEndian)
	}
}

func BenchmarkUnmarshal_GamePacket(b *testing.B) {
	data := []byte{0x4B, 0x96}
	var pkt GamePacket
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = nibble.Unmarshal(data, &pkt, nibble.LittleEndian)
	}
}

func BenchmarkMarshal_GamePacket(b *testing.B) {
	pkt := GamePacket{IsAlive: true, WeaponID: 5, TeamID: 2, Health: 300}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = nibble.Marshal(&pkt, nibble.LittleEndian)
	}
}

// ── named BE/LE API (0 allocs Unmarshal, 1 alloc Marshal for output buf) ──

func BenchmarkUnmarshalBE_TCPFlags(b *testing.B) {
	data := []byte{0x12}
	var flags TCPFlags
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = nibble.UnmarshalBE(data, &flags)
	}
}

func BenchmarkMarshalBE_TCPFlags(b *testing.B) {
	flags := TCPFlags{ACK: true, SYN: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = nibble.MarshalBE(&flags)
	}
}

func BenchmarkUnmarshalLE_GamePacket(b *testing.B) {
	data := []byte{0x4B, 0x96}
	var pkt GamePacket
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = nibble.UnmarshalLE(data, &pkt)
	}
}

func BenchmarkMarshalLE_GamePacket(b *testing.B) {
	pkt := GamePacket{IsAlive: true, WeaponID: 5, TeamID: 2, Health: 300}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = nibble.MarshalLE(&pkt)
	}
}

// ── MarshalInto: zero-alloc marshal with caller-supplied buffer ────────────

func BenchmarkMarshalInto_TCPFlags(b *testing.B) {
	flags := TCPFlags{ACK: true, SYN: true}
	buf := make([]byte, 1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = nibble.MarshalInto(buf, &flags, true)
	}
}

func BenchmarkMarshalInto_GamePacket(b *testing.B) {
	pkt := GamePacket{IsAlive: true, WeaponID: 5, TeamID: 2, Health: 300}
	buf := make([]byte, 2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = nibble.MarshalInto(buf, &pkt, false)
	}
}

// ── manual baselines ───────────────────────────────────────────────────────

func BenchmarkManual_Unmarshal_TCPFlags(b *testing.B) {
	data := []byte{0x12}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw := data[0]
		_ = TCPFlags{
			CWR: (raw>>7)&1 == 1,
			ECE: (raw>>6)&1 == 1,
			URG: (raw>>5)&1 == 1,
			ACK: (raw>>4)&1 == 1,
			PSH: (raw>>3)&1 == 1,
			RST: (raw>>2)&1 == 1,
			SYN: (raw>>1)&1 == 1,
			FIN: raw&1 == 1,
		}
	}
}

func BenchmarkManual_Marshal_TCPFlags(b *testing.B) {
	flags := TCPFlags{ACK: true, SYN: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out byte
		if flags.CWR {
			out |= 1 << 7
		}
		if flags.ECE {
			out |= 1 << 6
		}
		if flags.URG {
			out |= 1 << 5
		}
		if flags.ACK {
			out |= 1 << 4
		}
		if flags.PSH {
			out |= 1 << 3
		}
		if flags.RST {
			out |= 1 << 2
		}
		if flags.SYN {
			out |= 1 << 1
		}
		if flags.FIN {
			out |= 1
		}
		_ = out
	}
}

func BenchmarkManual_Unmarshal_GamePacket(b *testing.B) {
	data := []byte{0x4B, 0x96}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := uint16(data[0]) | uint16(data[1])<<8
		_ = GamePacket{
			IsAlive:  w&0x1 != 0,
			WeaponID: uint8((w >> 1) & 0xF),
			TeamID:   uint8((w >> 5) & 0x3),
			Health:   uint16((w >> 7) & 0x1FF),
		}
	}
}
