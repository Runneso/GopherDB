package semantic

type ResolvedConst struct {
	value    any
	exprType ExprType
}

func NewResolvedConst(value any, exprType ExprType) *ResolvedConst {
	return &ResolvedConst{
		value:    value,
		exprType: exprType,
	}
}

func (resolvedConst *ResolvedConst) Value() any {
	return resolvedConst.value
}

func (resolvedConst *ResolvedConst) ExprType() ExprType {
	return resolvedConst.exprType
}
