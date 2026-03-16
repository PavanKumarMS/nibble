package bitpack

import (
	"fmt"
	"reflect"
)

// Unmarshal decodes src into the struct pointed to by dst.
// dst must be a non-nil pointer to a struct whose fields carry `bits:"N"` tags.
// Fields are decoded in declaration order.
func Unmarshal(src []byte, dst any, opts ...Option) error {
	o := applyOptions(opts)

	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("bitpack: Unmarshal requires a non-nil pointer, got %T", dst)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("bitpack: Unmarshal requires a pointer to struct, got *%s", rv.Kind())
	}

	fields, err := parseFields(rv.Type())
	if err != nil {
		return err
	}

	need := (totalBits(fields) + 7) / 8
	if len(src) < need {
		return globalErr(
			fmt.Sprintf("need %d bytes, got %d", need, len(src)),
			ErrInsufficientData,
		)
	}

	pos := 0
	for _, fi := range fields {
		raw := readBits(src, pos, fi.BitWidth, o.bigEndian)
		pos += fi.BitWidth

		fv := rv.Field(fi.Index)
		if err := setField(fv, fi, raw); err != nil {
			return err
		}
	}
	return nil
}

// setField converts the raw uint64 bit pattern into the correct Go value and
// stores it in fv.
func setField(fv reflect.Value, fi fieldInfo, raw uint64) error {
	switch fi.Kind {
	case reflect.Bool:
		fv.SetBool(raw != 0)

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fv.SetUint(raw)

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Sign-extend using two's complement for the given bit width.
		signed := signExtend(raw, fi.BitWidth)
		fv.SetInt(signed)

	default:
		return fieldErr(fi.Name, fmt.Sprintf("unsupported kind %s", fi.Kind), ErrUnsupportedType)
	}
	return nil
}

// signExtend extends an n-bit two's-complement value to int64.
func signExtend(raw uint64, n int) int64 {
	if n == 64 {
		return int64(raw)
	}
	signBit := uint64(1) << (n - 1)
	if raw&signBit != 0 {
		// negative: fill upper bits with 1s
		mask := ^((signBit << 1) - 1)
		return int64(raw | mask)
	}
	return int64(raw)
}
