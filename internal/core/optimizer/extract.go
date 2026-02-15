package optimizer

import (
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/sql/semantic"
)

func extractEquality(pred semantic.ResolvedExpr) (*model.ColumnDefinition, any) {
	b, ok := pred.(*semantic.ResolvedBinaryExpr)
	if !ok || b.Op() != "=" {
		return nil, nil
	}
	if col, val, ok := tryExtractColVal(b.Left(), b.Right()); ok {
		return col, val
	}
	if col, val, ok := tryExtractColVal(b.Right(), b.Left()); ok {
		return col, val
	}
	return nil, nil
}

func tryExtractColVal(colExpr, valExpr semantic.ResolvedExpr) (*model.ColumnDefinition, any, bool) {
	col, ok := colExpr.(*semantic.ResolvedColumnRef)
	if !ok {
		return nil, nil, false
	}
	val, ok := valExpr.(*semantic.ResolvedConst)
	if !ok {
		return nil, nil, false
	}
	return col.Column(), val.Value(), true
}

func compareAny(a, b any) int {
	if av, ok := a.(int64); ok {
		return compareInt64(av, b.(int64))
	}
	if av, ok := a.(string); ok {
		return compareString(av, b.(string))
	}
	return 0
}

func compareInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
