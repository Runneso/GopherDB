package nodes

import (
	"strings"

	"GopherDB/internal/core/catalog/model"
)

type ProjectNode struct {
	child   PhysicalPlanNode
	columns []*model.ColumnDefinition
}

func NewProjectNode(child PhysicalPlanNode, columns []*model.ColumnDefinition) *ProjectNode {
	return &ProjectNode{
		child:   child,
		columns: columns,
	}
}

func (node *ProjectNode) Child() PhysicalPlanNode {
	return node.child
}

func (node *ProjectNode) Columns() []*model.ColumnDefinition {
	return node.columns
}

func (node *ProjectNode) DisplayName() string {
	names := make([]string, len(node.columns))
	for i, col := range node.columns {
		names[i] = col.Name()
	}
	return "Project(" + strings.Join(names, ",") + ")"
}

func (node *ProjectNode) Children() []PhysicalPlanNode {
	return []PhysicalPlanNode{node.child}
}
