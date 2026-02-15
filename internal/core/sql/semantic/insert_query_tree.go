package semantic

import "GopherDB/internal/core/catalog/model"

type InsertQueryTree struct {
	table   *model.TableDefinition
	columns []*model.ColumnDefinition
	values  []any
}

func NewInsertQueryTree(table *model.TableDefinition, columns []*model.ColumnDefinition, values []any) *InsertQueryTree {
	return &InsertQueryTree{
		table:   table,
		columns: columns,
		values:  values,
	}
}

func (query *InsertQueryTree) Type() QueryType {
	return QueryTypeInsert
}

func (query *InsertQueryTree) Table() *model.TableDefinition {
	return query.table
}

func (query *InsertQueryTree) Columns() []*model.ColumnDefinition {
	return query.columns
}

func (query *InsertQueryTree) Values() []any {
	return query.values
}
