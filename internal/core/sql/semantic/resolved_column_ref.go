package semantic

import "GopherDB/internal/core/catalog/model"

type ResolvedColumnRef struct {
	column   *model.ColumnDefinition
	exprType ExprType
}

func NewResolvedColumnRef(column *model.ColumnDefinition, exprType ExprType) *ResolvedColumnRef {
	return &ResolvedColumnRef{
		column:   column,
		exprType: exprType,
	}
}

func (ref *ResolvedColumnRef) Column() *model.ColumnDefinition {
	return ref.column
}

func (ref *ResolvedColumnRef) ExprType() ExprType {
	return ref.exprType
}
