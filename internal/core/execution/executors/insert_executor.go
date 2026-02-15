package executors

import (
	"GopherDB/internal/core/sql/semantic"
	"GopherDB/internal/core/storage"
)

type InsertExecutor struct {
	tableHeap *storage.TableHeap
	query     *semantic.InsertQueryTree
	executed  bool
}

func NewInsertExecutor(tableHeap *storage.TableHeap, query *semantic.InsertQueryTree) *InsertExecutor {
	return &InsertExecutor{tableHeap: tableHeap, query: query}
}

func (executor *InsertExecutor) Open() error {
	return nil
}

func (executor *InsertExecutor) Next() ([]any, error) {
	if executor.executed {
		return nil, nil
	}
	executor.executed = true

	_, err := executor.tableHeap.InsertRow(executor.query.Values())
	return nil, err
}

func (executor *InsertExecutor) Close() error {
	return nil
}
