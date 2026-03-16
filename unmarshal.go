package nibble

import (
	"fmt"
	"reflect"
)

// Unmarshal decodes src into the struct pointed to by dst.
// dst must be a non-nil pointer to a struct whose fields carry `bits:"N"` tags.
// Fields are decoded in declaration order.
//
// The struct schema is parsed and cached on the first call per type, so
// subsequent calls pay no reflection or string-parsing cost.
func Unmarshal(src []byte, dst any, opts ...Option) error {
	o := applyOptions(opts)

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
		if o.bigEndian {
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
			// Sign-extend using pre-computed masks — no per-call bit arithmetic.
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
