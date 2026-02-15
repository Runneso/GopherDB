package ast

type LiteralInt64Expr struct {
	value int64
}

func NewLiteralInt64Expr(value int64) *LiteralInt64Expr {
	return &LiteralInt64Expr{
		value: value,
	}
}

func (expr *LiteralInt64Expr) astNode() {}
func (expr *LiteralInt64Expr) expr()    {}

func (expr *LiteralInt64Expr) Value() int64 {
	return expr.value
}
