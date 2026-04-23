package nibble

import (
	"fmt"
	"reflect"
)

// Marshal encodes src into a new byte slice using the supplied byte-order
// option (default: LittleEndian). Allocates one []byte per call.
//
// For a zero-allocation hot path supply your own buffer via MarshalInto,
// or use MarshalBE / MarshalLE if you only need to avoid the options overhead.
func Marshal(src any, opts ...Option) ([]byte, error) {
	return marshalAlloc(src, parseBigEndian(opts))
}

// MarshalBE encodes src into a new byte slice using BigEndian (MSB-first)
// bit order. One allocation (the returned []byte).
func MarshalBE(src any) ([]byte, error) { return marshalAlloc(src, true) }

// MarshalLE encodes src into a new byte slice using LittleEndian (LSB-first)
// bit order. One allocation (the returned []byte).
func MarshalLE(src any) ([]byte, error) { return marshalAlloc(src, false) }

// MarshalInto encodes src into the caller-supplied dst buffer and returns the
// number of bytes written. dst must be at least Schema(src).ByteSize bytes.
// Zero allocations — use this in latency-critical hot paths with a pooled or
// stack-allocated buffer.
func MarshalInto(dst []byte, src any, bigEndian bool) (int, error) {
	rv, s, err := resolveStruct(src)
	if err != nil {
		return 0, err
	}
	if len(dst) < s.ByteSize {
		return 0, globalErr(
			fmt.Sprintf("buffer too small: need %d bytes, got %d", s.ByteSize, len(dst)),
			ErrInsufficientData,
		)
	}
	// Zero out only the bytes we will write so callers can safely reuse buffers.
	for i := 0; i < s.ByteSize; i++ {
		dst[i] = 0
	}
	return s.ByteSize, marshalFields(dst, rv, s, bigEndian)
}

// marshalAlloc allocates the output buffer then delegates to marshalFields.
func marshalAlloc(src any, bigEndian bool) ([]byte, error) {
	rv, s, err := resolveStruct(src)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, s.ByteSize)
	if err := marshalFields(buf, rv, s, bigEndian); err != nil {
		return nil, err
	}
	return buf, nil
}

// resolveStruct dereferences a pointer (if any) and looks up the schema.
func resolveStruct(src any) (reflect.Value, *schema, error) {
	rv := reflect.ValueOf(src)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return reflect.Value{}, nil, fmt.Errorf("nibble: Marshal requires a non-nil pointer, got nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, nil, fmt.Errorf("nibble: Marshal requires a struct or pointer to struct, got %T", src)
	}
	s, err := getSchema(rv.Type())
	if err != nil {
		return reflect.Value{}, nil, err
	}
	return rv, s, nil
}

// marshalFields writes all fields of rv into buf according to the schema.
func marshalFields(buf []byte, rv reflect.Value, s *schema, bigEndian bool) error {
	for i := range s.Fields {
		cf := &s.Fields[i]

		if cf.IsPad {
			continue
		}

		fv := fieldVal(rv, cf)

		var raw uint64
		switch cf.Kind {
		case reflect.Bool:
			if fv.Bool() {
				raw = 1
			}

		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v := fv.Uint()
			if cf.BitWidth < 64 && v > cf.Mask {
				return fieldErr(cf.Name,
					fmt.Sprintf("value %d overflows %d bits (max %d)", v, cf.BitWidth, cf.Mask),
					ErrFieldOverflow)
			}
			raw = v

		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := fv.Int()
			if cf.BitWidth < 64 && (v < cf.MinVal || v > cf.MaxVal) {
				return fieldErr(cf.Name,
					fmt.Sprintf("value %d out of range [%d, %d] for %d bits", v, cf.MinVal, cf.MaxVal, cf.BitWidth),
					ErrFieldOverflow)
			}
			raw = uint64(v) & cf.Mask

		default:
			return fieldErr(cf.Name, fmt.Sprintf("unsupported kind %s", cf.Kind), ErrUnsupportedType)
		}

		if bigEndian {
			writeBitsBE(buf, cf.BitOffset, cf.BitWidth, raw)
		} else {
			writeBitsLE(buf, cf.BitOffset, cf.BitWidth, raw)
		}
	}
	return nil
}
