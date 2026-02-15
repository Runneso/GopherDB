package ast

type InsertStmt struct {
	tableName *SqlIdent
	values    []Expr
}

func NewInsertStmt(tableName *SqlIdent, values []Expr) *InsertStmt {
	return &InsertStmt{
		tableName: tableName,
		values:    values,
	}
}

func (stmt *InsertStmt) astNode()   {}
func (stmt *InsertStmt) statement() {}

func (stmt *InsertStmt) TableName() *SqlIdent {
	return stmt.tableName
}

func (stmt *InsertStmt) Values() []Expr {
	return stmt.values
}
