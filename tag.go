package bitpack

import (
	"fmt"
	"reflect"
	"strconv"
)

// fieldInfo describes one struct field after tag parsing.
type fieldInfo struct {
	Name     string
	BitWidth int
	Index    int
	Kind     reflect.Kind
}

// maxBitsForKind returns the maximum number of bits for a reflect.Kind.
func maxBitsForKind(k reflect.Kind) int {
	switch k {
	case reflect.Bool:
		return 1
	case reflect.Int8, reflect.Uint8:
		return 8
	case reflect.Int16, reflect.Uint16:
		return 16
	case reflect.Int32, reflect.Uint32:
		return 32
	case reflect.Int64, reflect.Uint64:
		return 64
	default:
		return -1
	}
}

// isSupportedKind returns true for kinds we can pack.
func isSupportedKind(k reflect.Kind) bool {
	return maxBitsForKind(k) > 0
}

// parseFields inspects a struct type and returns a slice of fieldInfo,
// one per exported field that carries a `bits:"N"` tag.
func parseFields(t reflect.Type) ([]fieldInfo, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("bitpack: expected struct, got %s", t.Kind())
	}

	var fields []fieldInfo
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		tag, ok := f.Tag.Lookup("bits")
		if !ok {
			continue
		}

		bw, err := strconv.Atoi(tag)
		if err != nil || bw <= 0 {
			return nil, fieldErr(f.Name, fmt.Sprintf("invalid bits tag %q", tag), ErrBitWidthInvalid)
		}

		k := f.Type.Kind()
		if !isSupportedKind(k) {
			return nil, fieldErr(f.Name, fmt.Sprintf("type %s not supported", f.Type), ErrUnsupportedType)
		}

		max := maxBitsForKind(k)
		if bw > max {
			return nil, fieldErr(f.Name,
				fmt.Sprintf("bit width %d exceeds max %d for type %s", bw, max, f.Type),
				ErrBitWidthInvalid)
		}

		fields = append(fields, fieldInfo{
			Name:     f.Name,
			BitWidth: bw,
			Index:    i,
			Kind:     k,
		})
	}
	return fields, nil
}

// totalBits sums all field bit widths.
func totalBits(fields []fieldInfo) int {
	n := 0
	for _, f := range fields {
		n += f.BitWidth
	}
	return n
}
