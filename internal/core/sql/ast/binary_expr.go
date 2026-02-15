package ast

type BinaryExpr struct {
	op    string
	left  Expr
	right Expr
}

func NewBinaryExpr(op string, left, right Expr) *BinaryExpr {
	return &BinaryExpr{
		op:    op,
		left:  left,
		right: right,
	}
}

func (expr *BinaryExpr) astNode() {}
func (expr *BinaryExpr) expr()    {}

func (expr *BinaryExpr) Op() string {
	return expr.op
}

func (expr *BinaryExpr) Left() Expr {
	return expr.left
}

func (expr *BinaryExpr) Right() Expr {
	return expr.right
}
