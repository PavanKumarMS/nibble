// Package nibble provides declarative bit-level binary encoding and decoding
// using struct tags. Annotate struct fields with `bits:"N"` to describe how
// many bits each field occupies in the packed representation.
//
// Example:
//
//	type TCPFlags struct {
//	    CWR bool `bits:"1"`
//	    ECE bool `bits:"1"`
//	    URG bool `bits:"1"`
//	    ACK bool `bits:"1"`
//	    PSH bool `bits:"1"`
//	    RST bool `bits:"1"`
//	    SYN bool `bits:"1"`
//	    FIN bool `bits:"1"`
//	}
//
//	flags := TCPFlags{SYN: true, ACK: true}
//	data, _ := nibble.MarshalBE(&flags)
//	var out TCPFlags
//	nibble.UnmarshalBE(data, &out)
package nibble

// Option selects the bit-packing byte order.
//
// It is a concrete uint8 type (not a function) so that passing it as a
// variadic argument does not cause the options struct to escape to the heap.
type Option uint8

const (
	// LittleEndian (the default) packs the first struct field into the
	// least-significant bits of the first byte.
	LittleEndian Option = 0

	// BigEndian packs the first struct field into the most-significant bits
	// of the first byte (network byte order). Use for TCP, UDP, DNS, etc.
	BigEndian Option = 1
)

// parseBigEndian extracts the bigEndian bool from a variadic Option slice
// without allocating (no pointer-to-struct, no function calls through
// interface/func values).
func parseBigEndian(opts []Option) bool {
	for _, o := range opts {
		if o == BigEndian {
			return true
		}
	}
	return false
}

// FieldDiff records a single field that changed between two structs.
type FieldDiff struct {
	Field  string
	Before any
	After  any
}
