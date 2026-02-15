package executors

import "GopherDB/internal/core/catalog/model"

type ProjectExecutor struct {
	child     Executor
	positions []int32
	isOpen    bool
}

func NewProjectExecutor(child Executor, columns []*model.ColumnDefinition) *ProjectExecutor {
	positions := make([]int32, len(columns))
	for i, col := range columns {
		positions[i] = col.Position()
	}
	return &ProjectExecutor{child: child, positions: positions}
}

func (executor *ProjectExecutor) Open() error {
	if err := executor.child.Open(); err != nil {
		return err
	}
	executor.isOpen = true
	return nil
}

func (executor *ProjectExecutor) Next() ([]any, error) {
	if !executor.isOpen {
		return nil, ErrNotOpen
	}
	row, err := executor.child.Next()
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	out := make([]any, len(executor.positions))
	for i, pos := range executor.positions {
		out[i] = row[pos]
	}
	return out, nil
}

func (executor *ProjectExecutor) Close() error {
	executor.isOpen = false
	return executor.child.Close()
}
