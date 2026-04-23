package nibble

import (
	"fmt"
	"reflect"
	"strconv"
)

// fieldInfo describes one struct field after tag parsing.
type fieldInfo struct {
	Name      string
	BitWidth  int
	FieldPath []int        // reflect path to the struct field
	ArrayIdx  int          // -1 for scalars; >=0 for array element index
	Kind      reflect.Kind // kind of the scalar (or array element)
	IsPad     bool         // padding: consume bits but don't read/write
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

// parseFields inspects a struct type and returns a flat slice of fieldInfo.
// Nested structs are inlined; arrays are expanded per-element.
func parseFields(t reflect.Type) ([]fieldInfo, error) {
	return parseFieldsAt(t, nil, "")
}

// parseFieldsAt is the recursive core used by parseFields.
// pathPrefix is the reflect index path to the current struct within the root.
// namePrefix is the dot-separated name path for display (e.g. "Flags").
func parseFieldsAt(t reflect.Type, pathPrefix []int, namePrefix string) ([]fieldInfo, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("bitpack: expected struct, got %s", t.Kind())
	}

	var fields []fieldInfo
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag, hasTag := f.Tag.Lookup("bits")

		// bits:"-" — exclude field from packing entirely (no bits consumed).
		if hasTag && tag == "-" {
			continue
		}

		// Blank identifier padding: _ T `bits:"N"` — consume N bits without a name.
		if !f.IsExported() {
			if f.Name == "_" && hasTag {
				bw, err := strconv.Atoi(tag)
				if err != nil || bw <= 0 {
					return nil, fieldErr("_", fmt.Sprintf("invalid bits tag %q", tag), ErrBitWidthInvalid)
				}
				fields = append(fields, fieldInfo{
					Name:      "_",
					BitWidth:  bw,
					FieldPath: extendPath(pathPrefix, i),
					ArrayIdx:  -1,
					Kind:      f.Type.Kind(),
					IsPad:     true,
				})
			}
			continue
		}

		fullName := f.Name
		if namePrefix != "" {
			fullName = namePrefix + "." + f.Name
		}

		k := f.Type.Kind()

		// Nested struct without a bits tag — recurse and inline its fields.
		if k == reflect.Struct && !hasTag {
			sub, err := parseFieldsAt(f.Type, extendPath(pathPrefix, i), fullName)
			if err != nil {
				return nil, err
			}
			fields = append(fields, sub...)
			continue
		}

		if !hasTag {
			continue
		}

		bw, err := strconv.Atoi(tag)
		if err != nil || bw <= 0 {
			return nil, fieldErr(f.Name, fmt.Sprintf("invalid bits tag %q", tag), ErrBitWidthInvalid)
		}

		// Array field — expand into one entry per element.
		if k == reflect.Array {
			elemKind := f.Type.Elem().Kind()
			if !isSupportedKind(elemKind) {
				return nil, fieldErr(f.Name, fmt.Sprintf("array element type %s not supported", f.Type.Elem()), ErrUnsupportedType)
			}
			maxElem := maxBitsForKind(elemKind)
			if bw > maxElem {
				return nil, fieldErr(f.Name,
					fmt.Sprintf("bit width %d exceeds max %d for element type %s", bw, maxElem, f.Type.Elem()),
					ErrBitWidthInvalid)
			}
			path := extendPath(pathPrefix, i)
			for idx := 0; idx < f.Type.Len(); idx++ {
				fields = append(fields, fieldInfo{
					Name:      fmt.Sprintf("%s[%d]", fullName, idx),
					BitWidth:  bw,
					FieldPath: path,
					ArrayIdx:  idx,
					Kind:      elemKind,
				})
			}
			continue
		}

		// Scalar field.
		if !isSupportedKind(k) {
			return nil, fieldErr(f.Name, fmt.Sprintf("type %s not supported", f.Type), ErrUnsupportedType)
		}
		if bw > maxBitsForKind(k) {
			return nil, fieldErr(f.Name,
				fmt.Sprintf("bit width %d exceeds max %d for type %s", bw, maxBitsForKind(k), f.Type),
				ErrBitWidthInvalid)
		}
		fields = append(fields, fieldInfo{
			Name:      fullName,
			BitWidth:  bw,
			FieldPath: extendPath(pathPrefix, i),
			ArrayIdx:  -1,
			Kind:      k,
		})
	}
	return fields, nil
}

// extendPath returns a new slice with i appended to prefix.
func extendPath(prefix []int, i int) []int {
	path := make([]int, len(prefix)+1)
	copy(path, prefix)
	path[len(prefix)] = i
	return path
}

// totalBits sums all field bit widths.
func totalBits(fields []fieldInfo) int {
	n := 0
	for _, f := range fields {
		n += f.BitWidth
	}
	return n
}
