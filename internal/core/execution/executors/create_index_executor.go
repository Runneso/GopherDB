package executors

import (
	"fmt"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/memory/buffer"
	"GopherDB/internal/core/sql/semantic"
	"GopherDB/internal/core/storage"
)

type CreateIndexExecutor struct {
	root         string
	bufferPool   buffer.BufferPoolManager
	catalog      manager.CatalogManager
	indexManager *index.IndexManager
	query        *semantic.CreateIndexQueryTree
	executed     bool
}

func NewCreateIndexExecutor(root string, bufferPool buffer.BufferPoolManager, catalog manager.CatalogManager, indexManager *index.IndexManager, query *semantic.CreateIndexQueryTree) *CreateIndexExecutor {
	return &CreateIndexExecutor{
		root:         root,
		bufferPool:   bufferPool,
		catalog:      catalog,
		indexManager: indexManager,
		query:        query,
	}
}

func (executor *CreateIndexExecutor) Open() error {
	return nil
}

func (executor *CreateIndexExecutor) Next() ([]any, error) {
	if executor.executed {
		return nil, nil
	}
	executor.executed = true

	def, err := executor.catalog.CreateIndex(
		executor.query.IndexName().Text(),
		executor.query.Table().Name(),
		executor.query.Column().Name(),
		executor.query.IndexType(),
	)
	if err != nil {
		return nil, err
	}

	idx, err := executor.indexManager.GetOrCreate(def)
	if err != nil {
		return nil, err
	}

	tableHeap, err := storage.NewTableHeap(executor.root, executor.bufferPool, executor.catalog, executor.query.Table())
	if err != nil {
		return nil, err
	}

	columns, err := executor.catalog.GetColumns(executor.query.Table())
	if err != nil {
		return nil, err
	}

	colPos := -1
	for i, col := range columns {
		if col.Oid() == executor.query.Column().Oid() {
			colPos = i
			break
		}
	}
	if colPos < 0 {
		return nil, fmt.Errorf("index column not found in table: %s", executor.query.Column().Name())
	}

	tids, err := tableHeap.ScanTids()
	if err != nil {
		return nil, err
	}
	for _, tid := range tids {
		row, err := tableHeap.ReadRow(tid)
		if err != nil {
			return nil, err
		}
		if colPos >= len(row) {
			return nil, fmt.Errorf("index column position out of range: pos=%d", colPos)
		}
		if err := idx.Insert(row[colPos], tid); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (executor *CreateIndexExecutor) Close() error {
	return nil
}
