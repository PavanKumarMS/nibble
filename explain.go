package nibble

import (
	"fmt"
	"reflect"
	"strings"
)

// Explain returns a human-readable description of how src maps to the fields
// of the schema struct. schema must be a struct value or pointer-to-struct.
//
// Example output:
//
//	Byte 0 [00010010]:
//	  bit  7 → CWR:               false (0)
//	  bit  6 → ECE:               false (0)
//	  ...
func Explain(src []byte, schema any, opts ...Option) (string, error) {
	o := applyOptions(opts)

	rv := reflect.ValueOf(schema)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", fmt.Errorf("nibble: Explain requires a struct or pointer to struct")
	}

	s, err := getSchema(rv.Type())
	if err != nil {
		return "", err
	}

	if len(src) < s.ByteSize {
		return "", globalErr(
			fmt.Sprintf("need %d bytes, got %d", s.ByteSize, len(src)),
			ErrInsufficientData,
		)
	}

	// Build a per-bit annotation map: (byteIdx, bitInByte) → label + value.
	type annotation struct {
		fieldName string
		value     uint64
		bitWidth  int
	}
	type key struct{ byteIdx, bit int }
	annots := make(map[key]annotation, len(s.Fields)*8)

	for i := range s.Fields {
		cf := &s.Fields[i]
		raw := readBits(src, cf.BitOffset, cf.BitWidth, o.bigEndian)

		for j := 0; j < cf.BitWidth; j++ {
			streamPos := cf.BitOffset + j
			byteIdx := streamPos / 8
			bitIdx := streamPos % 8
			if o.bigEndian {
				bitIdx = 7 - bitIdx
			}
			annots[key{byteIdx, bitIdx}] = annotation{
				fieldName: cf.Name,
				value:     raw,
				bitWidth:  cf.BitWidth,
			}
		}
	}

	var sb strings.Builder
	for byteIdx := 0; byteIdx < s.ByteSize; byteIdx++ {
		b := src[byteIdx]
		sb.WriteString(fmt.Sprintf("Byte %d [%08b]:\n", byteIdx, b))

		// Print bits MSB→LSB (standard convention: bit 7 first).
		for bitPos := 7; bitPos >= 0; bitPos-- {
			a, ok := annots[key{byteIdx, bitPos}]
			if !ok {
				continue
			}
			bitVal := (b >> bitPos) & 1
			var valStr string
			if a.bitWidth == 1 {
				valStr = fmt.Sprintf("%v (%d)", bitVal != 0, bitVal)
			} else {
				valStr = fmt.Sprintf("%d (0x%x)", a.value, a.value)
			}
			sb.WriteString(fmt.Sprintf("  bit %2d → %-20s %s\n", bitPos, a.fieldName+":", valStr))
		}
	}
	return sb.String(), nil
}
