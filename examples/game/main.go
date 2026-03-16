// Command game demonstrates encoding and decoding a compact game-state packet
// with nibble.
package main

import (
	"fmt"
	"log"

	"github.com/PavanKumarMS/nibble"
)

// GamePacket is a 16-bit packed game-state message.
//
//	bit  0     : IsAlive  (1 bit)
//	bits 1-4   : WeaponID (4 bits, 0-15)
//	bits 5-6   : TeamID   (2 bits, 0-3)
//	bits 7-15  : Health   (9 bits, 0-511)
type GamePacket struct {
	IsAlive  bool   `bits:"1"`
	WeaponID uint8  `bits:"4"`
	TeamID   uint8  `bits:"2"`
	Health   uint16 `bits:"9"`
}

func main() {
	// ── Build a packet ────────────────────────────────────────────────────
	pkt := GamePacket{
		IsAlive:  true,
		WeaponID: 7,
		TeamID:   2,
		Health:   420,
	}

	if err := nibble.Validate(&pkt); err != nil {
		log.Fatalf("invalid packet: %v", err)
	}

	data, err := nibble.Marshal(&pkt, nibble.LittleEndian)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Encoded (%d bytes): %08b\n", len(data), data)

	// ── Decode it back ───────────────────────────────────────────────────
	var decoded GamePacket
	if err := nibble.Unmarshal(data, &decoded, nibble.LittleEndian); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded: %+v\n", decoded)

	// ── Show an annotated breakdown ───────────────────────────────────────
	explanation, err := nibble.Explain(data, GamePacket{}, nibble.LittleEndian)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nExplain:")
	fmt.Print(explanation)

	// ── Simulate a hit: player loses 100 HP ──────────────────────────────
	updated := decoded
	updated.Health -= 100

	diffs, err := nibble.Diff(decoded, updated)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nAfter taking 100 damage:")
	for _, d := range diffs {
		fmt.Printf("  %s: %v → %v\n", d.Field, d.Before, d.After)
	}

	// ── Demonstrate overflow detection ────────────────────────────────────
	bad := GamePacket{WeaponID: 20} // 20 > 15, overflows 4 bits
	if err := nibble.Validate(&bad); err != nil {
		fmt.Printf("\nValidation caught: %v\n", err)
	}
}
