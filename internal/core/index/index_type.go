package index

type IndexType int

const (
	HASH IndexType = iota
	BTREE
)

func (typ IndexType) Ordinal() int {
	return int(typ)
}

func IndexTypeFromOrdinal(ordinal int) IndexType {
	return IndexType(ordinal)
}
