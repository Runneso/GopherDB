package types

type IndexType int

const (
	IndexTypeHash IndexType = iota
	IndexTypeBTree
)

func (t IndexType) Ordinal() int {
	return int(t)
}

func IndexTypeFromOrdinal(ordinal int) IndexType {
	return IndexType(ordinal)
}
