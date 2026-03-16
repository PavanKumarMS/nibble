// Command tcp demonstrates parsing a TCP control-flags byte with bitpack.
package main

import (
	"fmt"
	"log"

	bitpack "github.com/pavankumarms/nibble"
)

// TCPFlags represents the 8-bit TCP control flags field.
// Fields are declared MSB-first so BigEndian encoding matches the wire format.
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

func main() {
	// ── Decode a raw TCP flags byte ──────────────────────────────────────
	// 0x12 = 0b00010010 → ACK + SYN set
	raw := []byte{0x12}

	var flags TCPFlags
	if err := bitpack.Unmarshal(raw, &flags, bitpack.BigEndian); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded flags: %+v\n", flags)
	// Output: Decoded flags: {CWR:false ECE:false URG:false ACK:true PSH:false RST:false SYN:true FIN:false}

	// ── Explain the byte ──────────────────────────────────────────────────
	explanation, err := bitpack.Explain(raw, TCPFlags{}, bitpack.BigEndian)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nExplain:")
	fmt.Print(explanation)

	// ── Encode back to bytes ──────────────────────────────────────────────
	encoded, err := bitpack.Marshal(&flags, bitpack.BigEndian)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nRe-encoded: 0x%02x\n", encoded[0])

	// ── Build a SYN packet and validate it ────────────────────────────────
	synPacket := TCPFlags{SYN: true}
	if err := bitpack.Validate(&synPacket); err != nil {
		log.Fatal(err)
	}
	synBytes, _ := bitpack.Marshal(&synPacket, bitpack.BigEndian)
	fmt.Printf("SYN-only byte: 0x%02x\n", synBytes[0]) // 0x02

	// ── Diff two packets ─────────────────────────────────────────────────
	synAck := TCPFlags{SYN: true, ACK: true}
	diffs, err := bitpack.Diff(synPacket, synAck)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nDiff SYN → SYN+ACK:\n")
	for _, d := range diffs {
		fmt.Printf("  %s: %v → %v\n", d.Field, d.Before, d.After)
	}
}
