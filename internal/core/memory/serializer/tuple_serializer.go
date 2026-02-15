package serializer

import "GopherDB/internal/core/memory/model"

type TupleSerializer interface {
	Serialize(any, model.DataType) (model.HeapTuple, error)
	Deserialize(model.HeapTuple) (any, error)
}
