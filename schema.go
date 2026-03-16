package nibble

import (
	"reflect"
	"sync"
)

// cachedField is a pre-computed, allocation-free representation of one struct
// field.  Every value that would otherwise be recalculated on every
// marshal/unmarshal call is computed exactly once here.
type cachedField struct {
	Name        string
	Index       int
	BitOffset   int          // cumulative bit position in the packed stream
	BitWidth    int
	Kind        reflect.Kind
	Mask        uint64 // (1<<BitWidth)-1  — extract / overflow-check mask
	IsSigned    bool
	SignBit     uint64 // 1<<(BitWidth-1)  — sign-extension test
	SignExtMask uint64 // ^Mask             — bits ORed in for negative values
	MinVal      int64  // lower bound for signed overflow check
	MaxVal      int64  // upper bound for signed overflow check
}

type schema struct {
	Fields   []cachedField
	ByteSize int // (totalBits+7)/8
}

// schemaCache maps reflect.Type → *schema.  Populated once per type.
var schemaCache sync.Map

// getSchema returns the cached schema for t, building and caching it on the
// first call for each type.
func getSchema(t reflect.Type) (*schema, error) {
	if v, ok := schemaCache.Load(t); ok {
		return v.(*schema), nil
	}

	fields, err := parseFields(t)
	if err != nil {
		return nil, err
	}

	cached := make([]cachedField, len(fields))
	offset := 0
	for i, f := range fields {
		var mask uint64
		if f.BitWidth == 64 {
			mask = ^uint64(0)
		} else {
			mask = (1 << f.BitWidth) - 1
		}

		cf := cachedField{
			Name:      f.Name,
			Index:     f.Index,
			BitOffset: offset,
			BitWidth:  f.BitWidth,
			Kind:      f.Kind,
			Mask:      mask,
		}

		switch f.Kind {
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			cf.IsSigned = true
			cf.SignBit = 1 << (f.BitWidth - 1)
			cf.SignExtMask = ^mask
			if f.BitWidth < 64 {
				cf.MinVal = -(int64(1) << (f.BitWidth - 1))
				cf.MaxVal = int64(1)<<(f.BitWidth-1) - 1
			} else {
				cf.MinVal = -1 << 63
				cf.MaxVal = 1<<63 - 1
			}
		}

		cached[i] = cf
		offset += f.BitWidth
	}

	s := &schema{
		Fields:   cached,
		ByteSize: (offset + 7) / 8,
	}
	schemaCache.Store(t, s)
	return s, nil
}
