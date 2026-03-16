package nibble

import (
	"fmt"
	"reflect"
)

// Marshal encodes the struct pointed to by src into a byte slice.
// src must be a non-nil pointer to a struct (or a struct value) whose fields
// carry `bits:"N"` tags. Fields are encoded in declaration order.
//
// The struct schema is parsed and cached on the first call per type.
func Marshal(src any, opts ...Option) ([]byte, error) {
	o := applyOptions(opts)

	rv := reflect.ValueOf(src)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("nibble: Marshal requires a non-nil pointer, got nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("nibble: Marshal requires a struct or pointer to struct, got %T", src)
	}

	s, err := getSchema(rv.Type())
	if err != nil {
		return nil, err
	}

	buf := make([]byte, s.ByteSize)

	for i := range s.Fields {
		cf := &s.Fields[i]
		fv := rv.Field(cf.Index)

		var raw uint64
		switch cf.Kind {
		case reflect.Bool:
			if fv.Bool() {
				raw = 1
			}

		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v := fv.Uint()
			if cf.BitWidth < 64 && v > cf.Mask {
				return nil, fieldErr(cf.Name,
					fmt.Sprintf("value %d overflows %d bits (max %d)", v, cf.BitWidth, cf.Mask),
					ErrFieldOverflow)
			}
			raw = v

		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := fv.Int()
			if cf.BitWidth < 64 && (v < cf.MinVal || v > cf.MaxVal) {
				return nil, fieldErr(cf.Name,
					fmt.Sprintf("value %d out of range [%d, %d] for %d bits", v, cf.MinVal, cf.MaxVal, cf.BitWidth),
					ErrFieldOverflow)
			}
			raw = uint64(v) & cf.Mask

		default:
			return nil, fieldErr(cf.Name, fmt.Sprintf("unsupported kind %s", cf.Kind), ErrUnsupportedType)
		}

		if o.bigEndian {
			writeBitsBE(buf, cf.BitOffset, cf.BitWidth, raw)
		} else {
			writeBitsLE(buf, cf.BitOffset, cf.BitWidth, raw)
		}
	}
	return buf, nil
}
