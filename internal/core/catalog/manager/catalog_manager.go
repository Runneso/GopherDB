package manager

import (
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/types"
)

type CatalogManager interface {
	CreateTable(name string, columns []*model.ColumnDefinition) (*model.TableDefinition, error)
	GetTable(tableName string) (*model.TableDefinition, error)
	ListTables() ([]*model.TableDefinition, error)
	GetColumns(table *model.TableDefinition) ([]*model.ColumnDefinition, error)
	GetColumn(table *model.TableDefinition, columnName string) (*model.ColumnDefinition, error)
	GetTypeByOid(oid int32) (*model.TypeDefinition, error)
	GetTypeByName(name string) (*model.TypeDefinition, error)
	UpdatePagesCount(table *model.TableDefinition, pagesCount int32) error
	CreateIndex(indexName, tableName, columnName string, indexType types.IndexType) (*model.IndexDefinition, error)
	GetIndex(indexName string) (*model.IndexDefinition, error)
	ListIndexes(table *model.TableDefinition) ([]*model.IndexDefinition, error)
}
