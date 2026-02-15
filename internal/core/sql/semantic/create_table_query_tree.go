package semantic

import "GopherDB/internal/core/sql/ast"

type CreateTableQueryTree struct {
	tableName *ast.SqlIdent
	columns   []*ResolvedCreateColumn
}

func NewCreateTableQueryTree(tableName *ast.SqlIdent, columns []*ResolvedCreateColumn) *CreateTableQueryTree {
	return &CreateTableQueryTree{
		tableName: tableName,
		columns:   columns,
	}
}

func (query *CreateTableQueryTree) Type() QueryType {
	return QueryTypeCreateTable
}

func (query *CreateTableQueryTree) TableName() *ast.SqlIdent {
	return query.tableName
}

func (query *CreateTableQueryTree) Columns() []*ResolvedCreateColumn {
	return query.columns
}
