package execution

import "GopherDB/internal/core/execution/executors"

type DefaultQueryExecutionEngine struct{}

func NewDefaultQueryExecutionEngine() *DefaultQueryExecutionEngine {
	return &DefaultQueryExecutionEngine{}
}

func (engine *DefaultQueryExecutionEngine) Execute(executor executors.Executor) ([][]any, error) {
	if err := executor.Open(); err != nil {
		return nil, err
	}
	defer executor.Close()

	var rows [][]any
	for {
		row, err := executor.Next()
		if err != nil {
			return nil, err
		}
		if row == nil {
			break
		}
		rows = append(rows, row)
	}

	return rows, nil
}
