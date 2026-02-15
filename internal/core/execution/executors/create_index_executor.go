package executors

import (
	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/sql/semantic"
)

type CreateIndexExecutor struct {
	catalog  manager.CatalogManager
	query    *semantic.CreateIndexQueryTree
	executed bool
}

func NewCreateIndexExecutor(catalog manager.CatalogManager, query *semantic.CreateIndexQueryTree) *CreateIndexExecutor {
	return &CreateIndexExecutor{catalog: catalog, query: query}
}

func (executor *CreateIndexExecutor) Open() error {
	return nil
}

func (executor *CreateIndexExecutor) Next() ([]any, error) {
	if executor.executed {
		return nil, nil
	}
	executor.executed = true

	_, err := executor.catalog.CreateIndex(
		executor.query.IndexName().Text(),
		executor.query.Table().Name(),
		executor.query.Column().Name(),
		executor.query.IndexType(),
	)
	return nil, err
}

func (executor *CreateIndexExecutor) Close() error {
	return nil
}
