package executors

import (
	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/sql/semantic"
)

type CreateTableExecutor struct {
	catalog  manager.CatalogManager
	query    *semantic.CreateTableQueryTree
	executed bool
}

func NewCreateTableExecutor(catalog manager.CatalogManager, query *semantic.CreateTableQueryTree) *CreateTableExecutor {
	return &CreateTableExecutor{catalog: catalog, query: query}
}

func (executor *CreateTableExecutor) Open() error {
	return nil
}

func (executor *CreateTableExecutor) Next() ([]any, error) {
	if executor.executed {
		return nil, nil
	}
	executor.executed = true

	var columns []*model.ColumnDefinition
	for i, c := range executor.query.Columns() {
		col, err := model.NewColumnDefinition(0, 0, c.TypeDef().Oid(), c.Name().Text(), int32(i))
		if err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	_, err := executor.catalog.CreateTable(executor.query.TableName().Text(), columns)
	return nil, err
}

func (executor *CreateTableExecutor) Close() error {
	return nil
}
