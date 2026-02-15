package execution

import "GopherDB/internal/core/execution/executors"

type QueryExecutionEngine interface {
	Execute(executor executors.Executor) ([][]any, error)
}
