package executors

import (
	"reflect"

	"GopherDB/internal/core/sql/semantic"
)

type evalFunc func(*FilterExecutor, semantic.ResolvedExpr, []any) any

type FilterExecutor struct {
	child     Executor
	predicate semantic.ResolvedExpr
	evalFuncs map[reflect.Type]evalFunc
	isOpen    bool
}

func NewFilterExecutor(child Executor, predicate semantic.ResolvedExpr) *FilterExecutor {
	executor := &FilterExecutor{child: child, predicate: predicate}
	executor.evalFuncs = map[reflect.Type]evalFunc{
		reflect.TypeOf((*semantic.ResolvedConst)(nil)):      evalConst,
		reflect.TypeOf((*semantic.ResolvedColumnRef)(nil)):  evalColumnRef,
		reflect.TypeOf((*semantic.ResolvedBinaryExpr)(nil)): evalBinary,
	}
	return executor
}

func (executor *FilterExecutor) Open() error {
	if err := executor.child.Open(); err != nil {
		return err
	}
	executor.isOpen = true
	return nil
}

func (executor *FilterExecutor) Next() ([]any, error) {
	if !executor.isOpen {
		return nil, ErrNotOpen
	}
	for {
		row, err := executor.child.Next()
		if err != nil {
			return nil, err
		}
		if row == nil {
			return nil, nil
		}
		if executor.evalBool(executor.predicate, row) {
			return row, nil
		}
	}
}

func (executor *FilterExecutor) Close() error {
	executor.isOpen = false
	return executor.child.Close()
}

func (executor *FilterExecutor) evalBool(expr semantic.ResolvedExpr, row []any) bool {
	v := executor.eval(expr, row)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func (executor *FilterExecutor) eval(expr semantic.ResolvedExpr, row []any) any {
	if fn, ok := executor.evalFuncs[reflect.TypeOf(expr)]; ok {
		return fn(executor, expr, row)
	}
	return nil
}

func evalConst(_ *FilterExecutor, expr semantic.ResolvedExpr, _ []any) any {
	return expr.(*semantic.ResolvedConst).Value()
}

func evalColumnRef(_ *FilterExecutor, expr semantic.ResolvedExpr, row []any) any {
	return row[expr.(*semantic.ResolvedColumnRef).Column().Position()]
}

func evalBinary(e *FilterExecutor, expr semantic.ResolvedExpr, row []any) any {
	b := expr.(*semantic.ResolvedBinaryExpr)
	op := b.Op()

	if op == "AND" {
		return e.evalBool(b.Left(), row) && e.evalBool(b.Right(), row)
	}
	if op == "OR" {
		return e.evalBool(b.Left(), row) || e.evalBool(b.Right(), row)
	}

	left := e.eval(b.Left(), row)
	right := e.eval(b.Right(), row)

	if op == "=" {
		return left == right
	}
	if op == "<>" {
		return left != right
	}

	cmp := compare(left, right)
	if op == "<" {
		return cmp < 0
	}
	if op == "<=" {
		return cmp <= 0
	}
	if op == ">" {
		return cmp > 0
	}
	if op == ">=" {
		return cmp >= 0
	}
	return false
}

func compare(a, b any) int {
	if ai, ok := a.(int64); ok {
		bi := b.(int64)
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0
	}
	if as, ok := a.(string); ok {
		bs := b.(string)
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0
	}
	return 0
}
