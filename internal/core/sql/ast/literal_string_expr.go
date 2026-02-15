package ast

type LiteralStringExpr struct {
	value string
}

func NewLiteralStringExpr(value string) *LiteralStringExpr {
	return &LiteralStringExpr{
		value: value,
	}
}

func (expr *LiteralStringExpr) astNode() {}
func (expr *LiteralStringExpr) expr()    {}

func (expr *LiteralStringExpr) Value() string {
	return expr.value
}
