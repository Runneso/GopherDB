package index

import "GopherDB/internal/core/catalog/model"

type Index interface {
	Definition() *model.IndexDefinition
	Insert(any, TID) error
	Search(any) ([]TID, error)
	RangeSearch(from any, fromInclusive bool, to any, toInclusive bool) ([]TID, error)
}
