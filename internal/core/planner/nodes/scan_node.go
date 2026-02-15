package nodes

import "GopherDB/internal/core/catalog/model"

type ScanNode struct {
	table *model.TableDefinition
}

func NewScanNode(table *model.TableDefinition) *ScanNode {
	return &ScanNode{
		table: table,
	}
}

func (node *ScanNode) Table() *model.TableDefinition {
	return node.table
}

func (node *ScanNode) DisplayName() string {
	return "Scan(" + node.table.Name() + ")"
}

func (node *ScanNode) Children() []LogicalPlanNode {
	return nil
}
