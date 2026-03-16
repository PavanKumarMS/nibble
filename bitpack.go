// Package bitpack provides declarative bit-level binary encoding and decoding
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
//	data, _ := bitpack.Marshal(&flags, bitpack.BigEndian)
//	var out TCPFlags
//	bitpack.Unmarshal(data, &out, bitpack.BigEndian)
package bitpack

// Option configures encoding/decoding behaviour.
type Option func(*options)

type options struct {
	bigEndian bool
}

func applyOptions(opts []Option) options {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// BigEndian packs the first struct field into the most-significant bits of the
// first byte (network byte order). This is the correct choice for most
// internet protocols (TCP, UDP, DNS, …).
var BigEndian Option = func(o *options) { o.bigEndian = true }

// LittleEndian (the default) packs the first struct field into the
// least-significant bits of the first byte.
var LittleEndian Option = func(o *options) { o.bigEndian = false }

// FieldDiff records a single field that changed between two structs.
type FieldDiff struct {
	Field  string
	Before any
	After  any
}
