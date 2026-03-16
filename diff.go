package nibble

import (
	"fmt"
	"reflect"
)

// Diff compares two structs of the same type and returns a slice of FieldDiff
// for every field whose value changed.  a and b must be structs (or pointers
// to structs) of identical types.
func Diff(a, b any) ([]FieldDiff, error) {
	ra := reflect.ValueOf(a)
	rb := reflect.ValueOf(b)

	if ra.Kind() == reflect.Ptr {
		ra = ra.Elem()
	}
	if rb.Kind() == reflect.Ptr {
		rb = rb.Elem()
	}

	if ra.Kind() != reflect.Struct || rb.Kind() != reflect.Struct {
		return nil, fmt.Errorf("bitpack: Diff requires two structs or pointers to structs")
	}
	if ra.Type() != rb.Type() {
		return nil, fmt.Errorf("bitpack: Diff requires both arguments to have the same type, got %s and %s",
			ra.Type(), rb.Type())
	}

	fields, err := parseFields(ra.Type())
	if err != nil {
		return nil, err
	}

	var diffs []FieldDiff
	for _, fi := range fields {
		va := ra.Field(fi.Index).Interface()
		vb := rb.Field(fi.Index).Interface()
		if !reflect.DeepEqual(va, vb) {
			diffs = append(diffs, FieldDiff{
				Field:  fi.Name,
				Before: va,
				After:  vb,
			})
		}
	}
	return diffs, nil
}
