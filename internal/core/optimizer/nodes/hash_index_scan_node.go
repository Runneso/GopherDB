package nodes

import (
	"fmt"

	"GopherDB/internal/core/catalog/model"
)

type HashIndexScanNode struct {
	table *model.TableDefinition
	index *model.IndexDefinition
	value any
}

func NewHashIndexScanNode(table *model.TableDefinition, index *model.IndexDefinition, value any) *HashIndexScanNode {
	return &HashIndexScanNode{
		table: table,
		index: index,
		value: value,
	}
}

func (node *HashIndexScanNode) Table() *model.TableDefinition {
	return node.table
}

func (node *HashIndexScanNode) Index() *model.IndexDefinition {
	return node.index
}

func (node *HashIndexScanNode) Value() any {
	return node.value
}

func (node *HashIndexScanNode) DisplayName() string {
	return fmt.Sprintf("HashIndexScan(%s, idx=%s, value=%v)", node.table.Name(), node.index.Name(), node.value)
}

func (node *HashIndexScanNode) Children() []PhysicalPlanNode {
	return nil
}
