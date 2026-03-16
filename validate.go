package nibble

import (
	"fmt"
	"reflect"
)

// Validate checks that every field value in src fits within its declared bit
// width. src must be a struct or a pointer to a struct.
// Returns nil if all values are in range, otherwise returns the first
// violation wrapped in ErrFieldOverflow.
func Validate(src any) error {
	rv := reflect.ValueOf(src)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return fmt.Errorf("nibble: Validate requires a non-nil value")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("nibble: Validate requires a struct or pointer to struct, got %T", src)
	}

	s, err := getSchema(rv.Type())
	if err != nil {
		return err
	}

	for i := range s.Fields {
		cf := &s.Fields[i]
		fv := rv.Field(cf.Index)

		switch cf.Kind {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v := fv.Uint()
			if cf.BitWidth < 64 && v > cf.Mask {
				return fieldErr(cf.Name,
					fmt.Sprintf("value %d overflows %d bits (max %d)", v, cf.BitWidth, cf.Mask),
					ErrFieldOverflow)
			}
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := fv.Int()
			if cf.BitWidth < 64 && (v < cf.MinVal || v > cf.MaxVal) {
				return fieldErr(cf.Name,
					fmt.Sprintf("value %d out of range [%d, %d] for %d bits", v, cf.MinVal, cf.MaxVal, cf.BitWidth),
					ErrFieldOverflow)
			}
		}
	}
	return nil
}
