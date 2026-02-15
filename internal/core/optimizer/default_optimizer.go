package optimizer

import (
	"reflect"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/optimizer/nodes"
	plannernodes "GopherDB/internal/core/planner/nodes"
	"GopherDB/internal/core/sql/semantic"
	"GopherDB/internal/core/types"
)

type nodeOptimizer func(plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode
type rangeBuilder func(*rangeInfo, any)

type DefaultOptimizer struct {
	catalog       manager.CatalogManager
	optimizers    map[reflect.Type]nodeOptimizer
	rangeBuilders map[string]rangeBuilder
	invertedOps   map[string]string
}

func NewDefaultOptimizer(catalog manager.CatalogManager) *DefaultOptimizer {
	optimizer := &DefaultOptimizer{
		catalog:     catalog,
		invertedOps: map[string]string{"<": ">", "<=": ">=", ">": "<", ">=": "<="},
	}
	optimizer.optimizers = map[reflect.Type]nodeOptimizer{
		reflect.TypeOf((*plannernodes.ExplainNode)(nil)):     optimizer.optimizeExplain,
		reflect.TypeOf((*plannernodes.CreateTableNode)(nil)): optimizer.optimizeCreateTable,
		reflect.TypeOf((*plannernodes.CreateIndexNode)(nil)): optimizer.optimizeCreateIndex,
		reflect.TypeOf((*plannernodes.InsertNode)(nil)):      optimizer.optimizeInsert,
		reflect.TypeOf((*plannernodes.ProjectNode)(nil)):     optimizer.optimizeProject,
		reflect.TypeOf((*plannernodes.FilterNode)(nil)):      optimizer.optimizeFilter,
		reflect.TypeOf((*plannernodes.ScanNode)(nil)):        optimizer.optimizeScan,
	}
	optimizer.rangeBuilders = map[string]rangeBuilder{
		"=":  buildEqRange,
		">":  buildGtRange,
		">=": buildGteRange,
		"<":  buildLtRange,
		"<=": buildLteRange,
	}
	return optimizer
}

func (optimizer *DefaultOptimizer) Optimize(plan plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	if fn, ok := optimizer.optimizers[reflect.TypeOf(plan)]; ok {
		return fn(plan)
	}
	return nil
}

func (optimizer *DefaultOptimizer) optimizeExplain(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewExplainNode(optimizer.Optimize(n.(*plannernodes.ExplainNode).Inner()))
}

func (optimizer *DefaultOptimizer) optimizeCreateTable(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewCreateTableNode(n.(*plannernodes.CreateTableNode).Query())
}

func (optimizer *DefaultOptimizer) optimizeCreateIndex(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewCreateIndexNode(n.(*plannernodes.CreateIndexNode).Query())
}

func (optimizer *DefaultOptimizer) optimizeInsert(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewInsertNode(n.(*plannernodes.InsertNode).Query())
}

func (optimizer *DefaultOptimizer) optimizeProject(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	p := n.(*plannernodes.ProjectNode)
	return nodes.NewProjectNode(optimizer.Optimize(p.Child()), p.Columns())
}

func (optimizer *DefaultOptimizer) optimizeFilter(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	f := n.(*plannernodes.FilterNode)
	scan, ok := f.Child().(*plannernodes.ScanNode)
	if !ok {
		return nodes.NewFilterNode(optimizer.Optimize(f.Child()), f.Predicate())
	}
	best := optimizer.chooseBestScan(scan.Table(), f.Predicate())
	return nodes.NewFilterNode(best, f.Predicate())
}

func (optimizer *DefaultOptimizer) optimizeScan(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewSeqScanNode(n.(*plannernodes.ScanNode).Table())
}

func (optimizer *DefaultOptimizer) chooseBestScan(table *model.TableDefinition, pred semantic.ResolvedExpr) nodes.PhysicalPlanNode {
	if node := optimizer.tryHashScan(table, pred); node != nil {
		return node
	}
	if node := optimizer.tryBTreeScan(table, pred); node != nil {
		return node
	}
	return nodes.NewSeqScanNode(table)
}

func (optimizer *DefaultOptimizer) tryHashScan(table *model.TableDefinition, pred semantic.ResolvedExpr) nodes.PhysicalPlanNode {
	col, val := extractEquality(pred)
	if col == nil {
		return nil
	}
	idx := optimizer.findIndex(table, col, types.IndexTypeHash)
	if idx == nil {
		return nil
	}
	return nodes.NewHashIndexScanNode(table, idx, val)
}

func (optimizer *DefaultOptimizer) tryBTreeScan(table *model.TableDefinition, pred semantic.ResolvedExpr) nodes.PhysicalPlanNode {
	rng := optimizer.extractRange(pred)
	if rng == nil {
		return nil
	}
	idx := optimizer.findIndex(table, rng.column, types.IndexTypeBTree)
	if idx == nil {
		return nil
	}
	return nodes.NewBTreeIndexScanNode(table, idx, rng.from, rng.fromInclusive, rng.to, rng.toInclusive)
}

func (optimizer *DefaultOptimizer) findIndex(table *model.TableDefinition, col *model.ColumnDefinition, t types.IndexType) *model.IndexDefinition {
	indexes, _ := optimizer.catalog.ListIndexes(table)
	for _, idx := range indexes {
		if idx.IndexType() == t && idx.ColumnOid() == col.Oid() {
			return idx
		}
	}
	return nil
}

func (optimizer *DefaultOptimizer) extractRange(pred semantic.ResolvedExpr) *rangeInfo {
	b, ok := pred.(*semantic.ResolvedBinaryExpr)
	if !ok {
		return nil
	}
	if b.Op() == "AND" {
		return optimizer.mergeAndRanges(b.Left(), b.Right())
	}
	return optimizer.extractSingleRange(b)
}

func (optimizer *DefaultOptimizer) mergeAndRanges(left, right semantic.ResolvedExpr) *rangeInfo {
	l, r := optimizer.extractRange(left), optimizer.extractRange(right)
	if l == nil {
		return r
	}
	if r == nil {
		return l
	}
	if l.column.Oid() != r.column.Oid() {
		return nil
	}
	return mergeRanges(l, r)
}

func (optimizer *DefaultOptimizer) extractSingleRange(b *semantic.ResolvedBinaryExpr) *rangeInfo {
	col, val, op := optimizer.extractColValOp(b)
	if col == nil {
		return nil
	}
	builder, ok := optimizer.rangeBuilders[op]
	if !ok {
		return nil
	}
	r := &rangeInfo{column: col}
	builder(r, val)
	return r
}

func (optimizer *DefaultOptimizer) extractColValOp(b *semantic.ResolvedBinaryExpr) (*model.ColumnDefinition, any, string) {
	if col, val, ok := tryExtractColVal(b.Left(), b.Right()); ok {
		return col, val, b.Op()
	}
	if col, val, ok := tryExtractColVal(b.Right(), b.Left()); ok {
		return col, val, optimizer.invertOp(b.Op())
	}
	return nil, nil, ""
}

func (optimizer *DefaultOptimizer) invertOp(op string) string {
	if inv, ok := optimizer.invertedOps[op]; ok {
		return inv
	}
	return op
}
