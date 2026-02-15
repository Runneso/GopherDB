package ast

type ColumnRefExpr struct {
	name *SqlIdent
}

func NewColumnRefExpr(name *SqlIdent) *ColumnRefExpr {
	return &ColumnRefExpr{
		name: name,
	}
}

func (expr *ColumnRefExpr) astNode() {}
func (expr *ColumnRefExpr) expr()    {}

func (expr *ColumnRefExpr) Name() *SqlIdent {
	return expr.name
}
