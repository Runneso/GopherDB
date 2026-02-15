package nodes

import (
	"fmt"

	"GopherDB/internal/core/catalog/model"
)

type BTreeIndexScanNode struct {
	table         *model.TableDefinition
	index         *model.IndexDefinition
	from          any
	fromInclusive bool
	to            any
	toInclusive   bool
}

func NewBTreeIndexScanNode(table *model.TableDefinition, index *model.IndexDefinition, from any, fromInclusive bool, to any, toInclusive bool) *BTreeIndexScanNode {
	return &BTreeIndexScanNode{
		table:         table,
		index:         index,
		from:          from,
		fromInclusive: fromInclusive,
		to:            to,
		toInclusive:   toInclusive,
	}
}

func (node *BTreeIndexScanNode) Table() *model.TableDefinition {
	return node.table
}

func (node *BTreeIndexScanNode) Index() *model.IndexDefinition {
	return node.index
}

func (node *BTreeIndexScanNode) From() any {
	return node.from
}

func (node *BTreeIndexScanNode) FromInclusive() bool {
	return node.fromInclusive
}

func (node *BTreeIndexScanNode) To() any {
	return node.to
}

func (node *BTreeIndexScanNode) ToInclusive() bool {
	return node.toInclusive
}

func (node *BTreeIndexScanNode) DisplayName() string {
	fromStr, toStr := "nil", "nil"
	if node.from != nil {
		fromStr = fmt.Sprintf("%v", node.from)
	}
	if node.to != nil {
		toStr = fmt.Sprintf("%v", node.to)
	}
	return fmt.Sprintf("BTreeIndexScan(%s, idx=%s, from=%s inc=%v, to=%s inc=%v)",
		node.table.Name(), node.index.Name(), fromStr, node.fromInclusive, toStr, node.toInclusive)
}

func (node *BTreeIndexScanNode) Children() []PhysicalPlanNode {
	return nil
}
