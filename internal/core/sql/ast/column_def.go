package ast

type ColumnDef struct {
	name     *SqlIdent
	typeName string
}

func NewColumnDef(name *SqlIdent, typeName string) *ColumnDef {
	return &ColumnDef{
		name:     name,
		typeName: typeName,
	}
}

func (def *ColumnDef) astNode() {}

func (def *ColumnDef) Name() *SqlIdent {
	return def.name
}

func (def *ColumnDef) TypeName() string {
	return def.typeName
}
