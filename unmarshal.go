package nibble

import (
	"fmt"
	"reflect"
)

// Unmarshal decodes src into the struct pointed to by dst using the supplied
// byte-order option (default: LittleEndian).
//
// For a zero-allocation hot path use UnmarshalBE or UnmarshalLE directly.
func Unmarshal(src []byte, dst any, opts ...Option) error {
	return unmarshal(src, dst, parseBigEndian(opts))
}

// UnmarshalBE decodes src into dst using BigEndian (MSB-first) bit order.
// Zero allocations.
func UnmarshalBE(src []byte, dst any) error { return unmarshal(src, dst, true) }

// UnmarshalLE decodes src into dst using LittleEndian (LSB-first) bit order.
// Zero allocations.
func UnmarshalLE(src []byte, dst any) error { return unmarshal(src, dst, false) }

// unmarshal is the allocation-free core shared by all public entry points.
func unmarshal(src []byte, dst any, bigEndian bool) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("nibble: Unmarshal requires a non-nil pointer, got %T", dst)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("nibble: Unmarshal requires a pointer to struct, got *%s", rv.Kind())
	}

	s, err := getSchema(rv.Type())
	if err != nil {
		return err
	}

	if len(src) < s.ByteSize {
		return globalErr(
			fmt.Sprintf("need %d bytes, got %d", s.ByteSize, len(src)),
			ErrInsufficientData,
		)
	}

	for i := range s.Fields {
		cf := &s.Fields[i]

		var raw uint64
		if bigEndian {
			raw = readBitsBE(src, cf.BitOffset, cf.BitWidth)
		} else {
			raw = readBitsLE(src, cf.BitOffset, cf.BitWidth)
		}

		fv := rv.Field(cf.Index)
		switch cf.Kind {
		case reflect.Bool:
			fv.SetBool(raw != 0)

		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fv.SetUint(raw)

		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var signed int64
			if raw&cf.SignBit != 0 {
				signed = int64(raw | cf.SignExtMask)
			} else {
				signed = int64(raw)
			}
			fv.SetInt(signed)

		default:
			return fieldErr(cf.Name, fmt.Sprintf("unsupported kind %s", cf.Kind), ErrUnsupportedType)
		}
	}
	return nil
}

// signExtend extends an n-bit two's-complement value to int64.
// Kept for use by any external callers; the hot path uses pre-computed masks.
func signExtend(raw uint64, n int) int64 {
	if n == 64 {
		return int64(raw)
	}
	signBit := uint64(1) << (n - 1)
	if raw&signBit != 0 {
		mask := ^((signBit << 1) - 1)
		return int64(raw | mask)
	}
	return int64(raw)
}
