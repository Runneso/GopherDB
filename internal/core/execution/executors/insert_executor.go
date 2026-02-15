package executors

import (
	"fmt"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/sql/semantic"
	"GopherDB/internal/core/storage"
)

type InsertExecutor struct {
	catalog      manager.CatalogManager
	indexManager *index.IndexManager
	tableHeap    *storage.TableHeap
	query        *semantic.InsertQueryTree
	executed     bool
}

func NewInsertExecutor(catalog manager.CatalogManager, indexManager *index.IndexManager, tableHeap *storage.TableHeap, query *semantic.InsertQueryTree) *InsertExecutor {
	return &InsertExecutor{catalog: catalog, indexManager: indexManager, tableHeap: tableHeap, query: query}
}

func (executor *InsertExecutor) Open() error {
	return nil
}

func (executor *InsertExecutor) Next() ([]any, error) {
	if executor.executed {
		return nil, nil
	}
	executor.executed = true

	tid, err := executor.tableHeap.InsertRow(executor.query.Values())
	if err != nil {
		return nil, err
	}

	indexes, err := executor.catalog.ListIndexes(executor.query.Table())
	if err != nil {
		return nil, err
	}
	if len(indexes) == 0 {
		return nil, nil
	}

	columns := executor.query.Columns()
	for _, idxDef := range indexes {
		idx, err := executor.indexManager.GetOrCreate(idxDef)
		if err != nil {
			return nil, err
		}

		colPos := -1
		for i, col := range columns {
			if col.Oid() == idxDef.ColumnOid() {
				colPos = i
				break
			}
		}
		if colPos < 0 || colPos >= len(executor.query.Values()) {
			return nil, fmt.Errorf("index column position not found for index %s", idxDef.Name())
		}

		if err := idx.Insert(executor.query.Values()[colPos], tid); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (executor *InsertExecutor) Close() error {
	return nil
}
