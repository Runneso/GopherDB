package nodes

import (
	"strings"

	"GopherDB/internal/core/catalog/model"
)

type ProjectNode struct {
	child   LogicalPlanNode
	columns []*model.ColumnDefinition
}

func NewProjectNode(child LogicalPlanNode, columns []*model.ColumnDefinition) *ProjectNode {
	return &ProjectNode{
		child:   child,
		columns: columns,
	}
}

func (node *ProjectNode) Child() LogicalPlanNode {
	return node.child
}

func (node *ProjectNode) Columns() []*model.ColumnDefinition {
	return node.columns
}

func (node *ProjectNode) DisplayName() string {
	names := make([]string, len(node.columns))
	for index, column := range node.columns {
		names[index] = column.Name()
	}
	return "Project(" + strings.Join(names, ",") + ")"
}

func (node *ProjectNode) Children() []LogicalPlanNode {
	return []LogicalPlanNode{node.child}
}
