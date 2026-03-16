package nibble

import (
	"fmt"
	"reflect"
)

// Validate checks that every field value in src fits within its declared bit
// width.  src must be a struct or a pointer to a struct.
// Returns nil if all values are in range, otherwise returns the first
// violation wrapped in ErrFieldOverflow.
func Validate(src any) error {
	rv := reflect.ValueOf(src)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return fmt.Errorf("bitpack: Validate requires a non-nil value")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("bitpack: Validate requires a struct or pointer to struct, got %T", src)
	}

	fields, err := parseFields(rv.Type())
	if err != nil {
		return err
	}

	for _, fi := range fields {
		fv := rv.Field(fi.Index)
		if _, err := getField(fv, fi); err != nil {
			return err
		}
	}
	return nil
}
