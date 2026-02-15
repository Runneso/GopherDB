package semantic

import (
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/sql/ast"
)

type CreateIndexQueryTree struct {
	indexName *ast.SqlIdent
	table     *model.TableDefinition
	column    *model.ColumnDefinition
	indexType index.IndexType
}

func NewCreateIndexQueryTree(indexName *ast.SqlIdent, table *model.TableDefinition, column *model.ColumnDefinition, indexType index.IndexType) *CreateIndexQueryTree {
	return &CreateIndexQueryTree{
		indexName: indexName,
		table:     table,
		column:    column,
		indexType: indexType,
	}
}

func (query *CreateIndexQueryTree) Type() QueryType {
	return QueryTypeCreateIndex
}

func (query *CreateIndexQueryTree) IndexName() *ast.SqlIdent {
	return query.indexName
}

func (query *CreateIndexQueryTree) Table() *model.TableDefinition {
	return query.table
}

func (query *CreateIndexQueryTree) Column() *model.ColumnDefinition {
	return query.column
}

func (query *CreateIndexQueryTree) IndexType() index.IndexType {
	return query.indexType
}
