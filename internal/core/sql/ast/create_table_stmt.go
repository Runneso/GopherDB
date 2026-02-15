package ast

type CreateTableStmt struct {
	tableName *SqlIdent
	columns   []*ColumnDef
}

func NewCreateTableStmt(tableName *SqlIdent, columns []*ColumnDef) *CreateTableStmt {
	return &CreateTableStmt{
		tableName: tableName,
		columns:   columns,
	}
}

func (stmt *CreateTableStmt) astNode()   {}
func (stmt *CreateTableStmt) statement() {}

func (stmt *CreateTableStmt) TableName() *SqlIdent {
	return stmt.tableName
}

func (stmt *CreateTableStmt) Columns() []*ColumnDef {
	return stmt.columns
}
