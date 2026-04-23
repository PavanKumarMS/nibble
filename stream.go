package nibble

import (
	"fmt"
	"io"
	"reflect"
)

// Reader decodes a sequence of structs from an io.Reader one at a time,
// reusing an internal buffer to avoid per-call allocations after the first.
type Reader struct {
	r         io.Reader
	bigEndian bool
	buf       []byte
}

// NewReader returns a Reader that decodes structs using the given byte order
// (default: LittleEndian). Use NewReaderBE / NewReaderLE to avoid the
// variadic options overhead.
func NewReader(r io.Reader, opts ...Option) *Reader {
	return &Reader{r: r, bigEndian: parseBigEndian(opts)}
}

// NewReaderBE returns a Reader that decodes using BigEndian bit order.
func NewReaderBE(r io.Reader) *Reader { return &Reader{r: r, bigEndian: true} }

// NewReaderLE returns a Reader that decodes using LittleEndian bit order.
func NewReaderLE(r io.Reader) *Reader { return &Reader{r: r, bigEndian: false} }

// Read decodes the next struct from the underlying io.Reader into dst.
// dst must be a non-nil pointer to a struct.
// Returns io.EOF if the stream is exhausted.
func (r *Reader) Read(dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("nibble: Reader.Read requires a non-nil pointer, got %T", dst)
	}
	if rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("nibble: Reader.Read requires a pointer to struct")
	}

	s, err := getSchema(rv.Elem().Type())
	if err != nil {
		return err
	}

	if cap(r.buf) < s.ByteSize {
		r.buf = make([]byte, s.ByteSize)
	}
	r.buf = r.buf[:s.ByteSize]

	if _, err := io.ReadFull(r.r, r.buf); err != nil {
		return err
	}
	return unmarshal(r.buf, dst, r.bigEndian)
}

// Writer encodes structs and writes them to an io.Writer one at a time,
// reusing an internal buffer to avoid per-call allocations after the first.
type Writer struct {
	w         io.Writer
	bigEndian bool
	buf       []byte
}

// NewWriter returns a Writer that encodes structs using the given byte order
// (default: LittleEndian). Use NewWriterBE / NewWriterLE to avoid the
// variadic options overhead.
func NewWriter(w io.Writer, opts ...Option) *Writer {
	return &Writer{w: w, bigEndian: parseBigEndian(opts)}
}

// NewWriterBE returns a Writer that encodes using BigEndian bit order.
func NewWriterBE(w io.Writer) *Writer { return &Writer{w: w, bigEndian: true} }

// NewWriterLE returns a Writer that encodes using LittleEndian bit order.
func NewWriterLE(w io.Writer) *Writer { return &Writer{w: w, bigEndian: false} }

// Write encodes src and writes the packed bytes to the underlying io.Writer.
// src must be a struct or a pointer to a struct.
func (w *Writer) Write(src any) error {
	rv := reflect.ValueOf(src)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return fmt.Errorf("nibble: Writer.Write requires a non-nil value")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("nibble: Writer.Write requires a struct or pointer to struct, got %T", src)
	}

	s, err := getSchema(rv.Type())
	if err != nil {
		return err
	}

	if cap(w.buf) < s.ByteSize {
		w.buf = make([]byte, s.ByteSize)
	}
	w.buf = w.buf[:s.ByteSize]

	if _, err := MarshalInto(w.buf, src, w.bigEndian); err != nil {
		return err
	}
	_, err = w.w.Write(w.buf)
	return err
}
