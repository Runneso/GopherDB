package ast

type ExplainStmt struct {
	inner Statement
}

func NewExplainStmt(inner Statement) *ExplainStmt {
	return &ExplainStmt{
		inner: inner,
	}
}

func (stmt *ExplainStmt) astNode()   {}
func (stmt *ExplainStmt) statement() {}

func (stmt *ExplainStmt) Inner() Statement {
	return stmt.inner
}
