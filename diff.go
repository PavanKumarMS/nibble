package nibble

import (
	"fmt"
	"reflect"
)

// Diff compares two structs of the same type and returns a slice of FieldDiff
// for every field whose value changed. a and b must be structs (or pointers
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
		return nil, fmt.Errorf("nibble: Diff requires two structs or pointers to structs")
	}
	if ra.Type() != rb.Type() {
		return nil, fmt.Errorf("nibble: Diff requires both arguments to have the same type, got %s and %s",
			ra.Type(), rb.Type())
	}

	s, err := getSchema(ra.Type())
	if err != nil {
		return nil, err
	}

	var diffs []FieldDiff
	for i := range s.Fields {
		cf := &s.Fields[i]
		if cf.IsPad {
			continue
		}
		va := fieldVal(ra, cf).Interface()
		vb := fieldVal(rb, cf).Interface()
		if !reflect.DeepEqual(va, vb) {
			diffs = append(diffs, FieldDiff{
				Field:  cf.Name,
				Before: va,
				After:  vb,
			})
		}
	}
	return diffs, nil
}
