package ast

import "GopherDB/internal/core/types"

type CreateIndexStmt struct {
	indexName  *SqlIdent
	tableName  *SqlIdent
	columnName *SqlIdent
	indexType  types.IndexType
}

func NewCreateIndexStmt(indexName, tableName, columnName *SqlIdent, indexType types.IndexType) *CreateIndexStmt {
	return &CreateIndexStmt{
		indexName:  indexName,
		tableName:  tableName,
		columnName: columnName,
		indexType:  indexType,
	}
}

func (stmt *CreateIndexStmt) astNode()   {}
func (stmt *CreateIndexStmt) statement() {}

func (stmt *CreateIndexStmt) IndexName() *SqlIdent {
	return stmt.indexName
}

func (stmt *CreateIndexStmt) TableName() *SqlIdent {
	return stmt.tableName
}

func (stmt *CreateIndexStmt) ColumnName() *SqlIdent {
	return stmt.columnName
}

func (stmt *CreateIndexStmt) IndexType() types.IndexType {
	return stmt.indexType
}
