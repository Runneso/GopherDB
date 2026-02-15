package semantic

import (
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/sql/ast"
)

type ResolvedCreateColumn struct {
	name    *ast.SqlIdent
	typeDef *model.TypeDefinition
}

func NewResolvedCreateColumn(name *ast.SqlIdent, typeDef *model.TypeDefinition) *ResolvedCreateColumn {
	return &ResolvedCreateColumn{
		name:    name,
		typeDef: typeDef,
	}
}

func (column *ResolvedCreateColumn) Name() *ast.SqlIdent {
	return column.name
}

func (column *ResolvedCreateColumn) TypeDef() *model.TypeDefinition {
	return column.typeDef
}
