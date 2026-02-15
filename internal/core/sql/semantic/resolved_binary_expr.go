package semantic

type ResolvedBinaryExpr struct {
	op       string
	left     ResolvedExpr
	right    ResolvedExpr
	exprType ExprType
}

func NewResolvedBinaryExpr(op string, left, right ResolvedExpr, exprType ExprType) *ResolvedBinaryExpr {
	return &ResolvedBinaryExpr{
		op:       op,
		left:     left,
		right:    right,
		exprType: exprType,
	}
}

func (expr *ResolvedBinaryExpr) Op() string {
	return expr.op
}

func (expr *ResolvedBinaryExpr) Left() ResolvedExpr {
	return expr.left
}

func (expr *ResolvedBinaryExpr) Right() ResolvedExpr {
	return expr.right
}

func (expr *ResolvedBinaryExpr) ExprType() ExprType {
	return expr.exprType
}
