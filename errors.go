package nibble

import (
	"errors"
	"fmt"
)

// Sentinel errors for wrapping/unwrapping with errors.Is.
var (
	ErrFieldOverflow    = errors.New("field value exceeds bit width")
	ErrInsufficientData = errors.New("insufficient data")
	ErrUnsupportedType  = errors.New("unsupported field type")
	ErrBitWidthInvalid  = errors.New("invalid bit width for field type")
)

// BitpackError is a structured error that carries field context.
type BitpackError struct {
	Field string
	Err   error
	Msg   string
}

func (e *BitpackError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("bitpack: field %q: %s: %v", e.Field, e.Msg, e.Err)
	}
	return fmt.Sprintf("bitpack: %s: %v", e.Msg, e.Err)
}

func (e *BitpackError) Unwrap() error { return e.Err }

func fieldErr(field, msg string, sentinel error) error {
	return &BitpackError{Field: field, Msg: msg, Err: sentinel}
}

func globalErr(msg string, sentinel error) error {
	return &BitpackError{Msg: msg, Err: sentinel}
}
