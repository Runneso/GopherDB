package nodes

import "GopherDB/internal/core/catalog/model"

type SeqScanNode struct {
	table *model.TableDefinition
}

func NewSeqScanNode(table *model.TableDefinition) *SeqScanNode {
	return &SeqScanNode{
		table: table,
	}
}

func (node *SeqScanNode) Table() *model.TableDefinition {
	return node.table
}

func (node *SeqScanNode) DisplayName() string {
	return "SeqScan(" + node.table.Name() + ")"
}

func (node *SeqScanNode) Children() []PhysicalPlanNode {
	return nil
}
