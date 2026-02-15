package semantic

type ExprType int

const (
	ExprTypeInt64 ExprType = iota
	ExprTypeVarchar
	ExprTypeBool
)
