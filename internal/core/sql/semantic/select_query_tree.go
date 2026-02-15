package semantic

import "GopherDB/internal/core/catalog/model"

type SelectQueryTree struct {
	table         *model.TableDefinition
	targetColumns []*model.ColumnDefinition
	filter        ResolvedExpr
}

func NewSelectQueryTree(table *model.TableDefinition, targetColumns []*model.ColumnDefinition, filter ResolvedExpr) *SelectQueryTree {
	return &SelectQueryTree{
		table:         table,
		targetColumns: targetColumns,
		filter:        filter,
	}
}

func (query *SelectQueryTree) Type() QueryType {
	return QueryTypeSelect
}

func (query *SelectQueryTree) Table() *model.TableDefinition {
	return query.table
}

func (query *SelectQueryTree) TargetColumns() []*model.ColumnDefinition {
	return query.targetColumns
}

func (query *SelectQueryTree) Filter() ResolvedExpr {
	return query.filter
}
