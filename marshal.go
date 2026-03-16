package nibble

import (
	"fmt"
	"reflect"
)

// Marshal encodes the struct pointed to by src into a byte slice.
// src must be a non-nil pointer to a struct (or a struct value) whose fields
// carry `bits:"N"` tags.  Fields are encoded in declaration order.
func Marshal(src any, opts ...Option) ([]byte, error) {
	o := applyOptions(opts)

	rv := reflect.ValueOf(src)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("bitpack: Marshal requires a non-nil pointer, got nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("bitpack: Marshal requires a struct or pointer to struct, got %T", src)
	}

	fields, err := parseFields(rv.Type())
	if err != nil {
		return nil, err
	}

	total := totalBits(fields)
	buf := make([]byte, (total+7)/8)

	pos := 0
	for _, fi := range fields {
		raw, err := getField(rv.Field(fi.Index), fi)
		if err != nil {
			return nil, err
		}
		writeBits(buf, pos, fi.BitWidth, raw, o.bigEndian)
		pos += fi.BitWidth
	}
	return buf, nil
}

// getField extracts the uint64 bit pattern from a struct field value.
func getField(fv reflect.Value, fi fieldInfo) (uint64, error) {
	switch fi.Kind {
	case reflect.Bool:
		if fv.Bool() {
			return 1, nil
		}
		return 0, nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v := fv.Uint()
		maxVal := uint64((1 << fi.BitWidth) - 1)
		if fi.BitWidth < 64 && v > maxVal {
			return 0, fieldErr(fi.Name,
				fmt.Sprintf("value %d overflows %d bits (max %d)", v, fi.BitWidth, maxVal),
				ErrFieldOverflow)
		}
		return v, nil

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v := fv.Int()
		// For signed N-bit field: valid range is [-2^(N-1), 2^(N-1)-1].
		if fi.BitWidth < 64 {
			minVal := -(int64(1) << (fi.BitWidth - 1))
			maxVal := int64(1)<<(fi.BitWidth-1) - 1
			if v < minVal || v > maxVal {
				return 0, fieldErr(fi.Name,
					fmt.Sprintf("value %d out of range [%d, %d] for %d bits", v, minVal, maxVal, fi.BitWidth),
					ErrFieldOverflow)
			}
		}
		// Mask to the bit width so only the low N bits are written.
		mask := uint64((1 << fi.BitWidth) - 1)
		if fi.BitWidth == 64 {
			mask = ^uint64(0)
		}
		return uint64(v) & mask, nil

	default:
		return 0, fieldErr(fi.Name, fmt.Sprintf("unsupported kind %s", fi.Kind), ErrUnsupportedType)
	}
}
