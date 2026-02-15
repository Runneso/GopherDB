package semantic

type QueryTree interface {
	Type() QueryType
}

type ResolvedExpr interface {
	ExprType() ExprType
}
