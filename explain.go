package bitpack

import (
	"fmt"
	"reflect"
	"strings"
)

// Explain returns a human-readable description of how src maps to the fields
// of the schema struct.  schema must be a struct value or pointer-to-struct
// of the same type that was used to produce src.
//
// Example output:
//
//	Byte 0 [10110011]:
//	  bit  7 → CWR: true  (1)
//	  bit  6 → ECE: false (0)
//	  ...
func Explain(src []byte, schema any, opts ...Option) (string, error) {
	o := applyOptions(opts)

	rv := reflect.ValueOf(schema)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", fmt.Errorf("bitpack: Explain requires a struct or pointer to struct")
	}

	fields, err := parseFields(rv.Type())
	if err != nil {
		return "", err
	}

	need := (totalBits(fields) + 7) / 8
	if len(src) < need {
		return "", globalErr(
			fmt.Sprintf("need %d bytes, got %d", need, len(src)),
			ErrInsufficientData,
		)
	}

	// Build a per-bit annotation: bitAnnotation[byteIdx][bitInByte] = label
	type annotation struct {
		fieldName string
		value     uint64
		bitWidth  int // total width of this field (for display context)
	}

	// maps (byteIndex, bitInByte) → annotation
	type key struct{ byteIdx, bit int }
	annots := make(map[key]annotation)

	pos := 0
	for _, fi := range fields {
		raw := readBits(src, pos, fi.BitWidth, o.bigEndian)
		for i := 0; i < fi.BitWidth; i++ {
			streamPos := pos + i
			byteIdx := streamPos / 8
			bitIdx := streamPos % 8
			if o.bigEndian {
				bitIdx = 7 - bitIdx
			}
			annots[key{byteIdx, bitIdx}] = annotation{
				fieldName: fi.Name,
				value:     raw,
				bitWidth:  fi.BitWidth,
			}
		}
		pos += fi.BitWidth
	}

	totalBytes := (totalBits(fields) + 7) / 8
	var sb strings.Builder

	for byteIdx := 0; byteIdx < totalBytes; byteIdx++ {
		b := src[byteIdx]
		sb.WriteString(fmt.Sprintf("Byte %d [%08b]:\n", byteIdx, b))

		// Print bits in order: MSB (bit 7) down to LSB (bit 0) – standard convention.
		for bitPos := 7; bitPos >= 0; bitPos-- {
			k := key{byteIdx, bitPos}
			a, ok := annots[k]
			if !ok {
				continue
			}

			bitVal := (b >> bitPos) & 1

			// Format the field value in a human-readable way.
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
