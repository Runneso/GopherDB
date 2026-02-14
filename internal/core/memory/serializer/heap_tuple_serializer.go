package serializer

import (
	"GopherDB/internal/core/memory/model"
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf8"
)

var (
	ErrNilType         = errors.New("type is zero/invalid")
	ErrNilValue        = errors.New("value is nil")
	ErrTupleDataNil    = errors.New("tuple data is nil")
	ErrUnsupportedType = errors.New("unsupported data type")
	ErrTypeMismatch    = errors.New("value type mismatch")
	ErrVarcharTooLong  = errors.New("varchar exceeds 255 bytes")
	ErrBadInt64Size    = errors.New("int64 must be exactly 8 bytes")
	ErrVarcharEmpty    = errors.New("varchar payload is empty, missing length byte")
	ErrVarcharLenMis   = errors.New("varchar length mismatch")
	ErrInvalidEncoding = errors.New("string is not utf-8")
)

const (
	int64Bytes       = 8
	maxVarCharLength = 255
)

type HeapTupleSerializer struct {
	serializers   map[model.DataType]func(any) (model.HeapTuple, error)
	deserializers map[model.DataType]func([]byte) (any, error)
}

func NewHeapTupleSerializer() *HeapTupleSerializer {
	heapTupleSerializer := &HeapTupleSerializer{
		serializers:   make(map[model.DataType]func(any) (model.HeapTuple, error), 2),
		deserializers: make(map[model.DataType]func([]byte) (any, error), 2),
	}

	heapTupleSerializer.serializers[model.INT64] = func(v any) (model.HeapTuple, error) {
		n, ok := v.(int64)
		if !ok {
			return model.HeapTuple{}, fmt.Errorf("%w: INT64 expects number, got %T", ErrTypeMismatch, v)
		}
		buf := make([]byte, int64Bytes)
		binary.BigEndian.PutUint64(buf, uint64(n))
		return model.NewHeapTuple(buf, model.INT64), nil
	}

	heapTupleSerializer.deserializers[model.INT64] = func(data []byte) (any, error) {
		if len(data) != int64Bytes {
			return nil, fmt.Errorf("%w: got=%d", ErrBadInt64Size, len(data))
		}
		return int64(binary.BigEndian.Uint64(data)), nil
	}

	heapTupleSerializer.serializers[model.VARCHAR] = func(value any) (model.HeapTuple, error) {
		str, ok := value.(string)
		if !ok {
			return model.HeapTuple{}, fmt.Errorf("%w: VARCHAR expects string, got %T", ErrTypeMismatch, value)
		}
		b := []byte(str)
		if len(b) > maxVarCharLength {
			return model.HeapTuple{}, fmt.Errorf("%w: %d", ErrVarcharTooLong, len(b))
		}
		out := make([]byte, 1+len(b))
		out[0] = byte(len(b))
		copy(out[1:], b)
		return model.NewHeapTuple(out, model.VARCHAR), nil
	}

	heapTupleSerializer.deserializers[model.VARCHAR] = func(data []byte) (any, error) {
		if len(data) == 0 {
			return nil, ErrVarcharEmpty
		}
		n := int(data[0])
		if len(data) != 1+n {
			return nil, fmt.Errorf("%w: header=%d bytes=%d", ErrVarcharLenMis, n, len(data)-1)
		}
		payload := data[1 : 1+n]
		if !utf8.Valid(payload) {
			return nil, ErrInvalidEncoding
		}
		return string(payload), nil
	}

	return heapTupleSerializer
}

func (heapTupleSerializer *HeapTupleSerializer) Serialize(value any, dt model.DataType) (model.HeapTuple, error) {
	if dt == 0 {
		return model.HeapTuple{}, ErrNilType
	}

	if value == nil {
		return model.HeapTuple{}, ErrNilValue
	}

	handler, exist := heapTupleSerializer.serializers[dt]
	if !exist {
		return model.HeapTuple{}, fmt.Errorf("%w: %v", ErrUnsupportedType, dt)
	}
	return handler(value)
}

func (heapTupleSerializer *HeapTupleSerializer) Deserialize(tuple model.HeapTuple) (any, error) {
	data := tuple.Data()
	if data == nil {
		return nil, ErrTupleDataNil
	}

	dt := tuple.Type()
	if dt == 0 {
		return nil, ErrNilType
	}

	handler, exist := heapTupleSerializer.deserializers[dt]
	if !exist {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedType, dt)
	}

	return handler(data)
}
