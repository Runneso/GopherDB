package ast

type SelectStmt struct {
	selectAll bool
	columns   []*SqlIdent
	tableName *SqlIdent
	where     Expr
}

func NewSelectStmt(selectAll bool, columns []*SqlIdent, tableName *SqlIdent, where Expr) *SelectStmt {
	return &SelectStmt{
		selectAll: selectAll,
		columns:   columns,
		tableName: tableName,
		where:     where,
	}
}

func (stmt *SelectStmt) astNode()   {}
func (stmt *SelectStmt) statement() {}

func (stmt *SelectStmt) SelectAll() bool {
	return stmt.selectAll
}

func (stmt *SelectStmt) Columns() []*SqlIdent {
	return stmt.columns
}

func (stmt *SelectStmt) TableName() *SqlIdent {
	return stmt.tableName
}

func (stmt *SelectStmt) Where() Expr {
	return stmt.where
}
