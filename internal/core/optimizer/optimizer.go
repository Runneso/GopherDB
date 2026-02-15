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

type Optimizer struct {
	catalog       manager.CatalogManager
	optimizers    map[reflect.Type]nodeOptimizer
	rangeBuilders map[string]rangeBuilder
	invertedOps   map[string]string
}

func NewOptimizer(catalog manager.CatalogManager) *Optimizer {
	o := &Optimizer{
		catalog:     catalog,
		invertedOps: map[string]string{"<": ">", "<=": ">=", ">": "<", ">=": "<="},
	}
	o.optimizers = map[reflect.Type]nodeOptimizer{
		reflect.TypeOf((*plannernodes.ExplainNode)(nil)):     o.optimizeExplain,
		reflect.TypeOf((*plannernodes.CreateTableNode)(nil)): o.optimizeCreateTable,
		reflect.TypeOf((*plannernodes.CreateIndexNode)(nil)): o.optimizeCreateIndex,
		reflect.TypeOf((*plannernodes.InsertNode)(nil)):      o.optimizeInsert,
		reflect.TypeOf((*plannernodes.ProjectNode)(nil)):     o.optimizeProject,
		reflect.TypeOf((*plannernodes.FilterNode)(nil)):      o.optimizeFilter,
		reflect.TypeOf((*plannernodes.ScanNode)(nil)):        o.optimizeScan,
	}
	o.rangeBuilders = map[string]rangeBuilder{
		"=":  buildEqRange,
		">":  buildGtRange,
		">=": buildGteRange,
		"<":  buildLtRange,
		"<=": buildLteRange,
	}
	return o
}

func (o *Optimizer) Optimize(plan plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	if fn, ok := o.optimizers[reflect.TypeOf(plan)]; ok {
		return fn(plan)
	}
	return nil
}

func (o *Optimizer) optimizeExplain(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewExplainNode(o.Optimize(n.(*plannernodes.ExplainNode).Inner()))
}

func (o *Optimizer) optimizeCreateTable(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewCreateTableNode(n.(*plannernodes.CreateTableNode).Query())
}

func (o *Optimizer) optimizeCreateIndex(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewCreateIndexNode(n.(*plannernodes.CreateIndexNode).Query())
}

func (o *Optimizer) optimizeInsert(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewInsertNode(n.(*plannernodes.InsertNode).Query())
}

func (o *Optimizer) optimizeProject(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	p := n.(*plannernodes.ProjectNode)
	return nodes.NewProjectNode(o.Optimize(p.Child()), p.Columns())
}

func (o *Optimizer) optimizeFilter(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	f := n.(*plannernodes.FilterNode)
	scan, ok := f.Child().(*plannernodes.ScanNode)
	if !ok {
		return nodes.NewFilterNode(o.Optimize(f.Child()), f.Predicate())
	}
	best := o.chooseBestScan(scan.Table(), f.Predicate())
	return nodes.NewFilterNode(best, f.Predicate())
}

func (o *Optimizer) optimizeScan(n plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode {
	return nodes.NewSeqScanNode(n.(*plannernodes.ScanNode).Table())
}

func (o *Optimizer) chooseBestScan(table *model.TableDefinition, pred semantic.ResolvedExpr) nodes.PhysicalPlanNode {
	if node := o.tryHashScan(table, pred); node != nil {
		return node
	}
	if node := o.tryBTreeScan(table, pred); node != nil {
		return node
	}
	return nodes.NewSeqScanNode(table)
}

func (o *Optimizer) tryHashScan(table *model.TableDefinition, pred semantic.ResolvedExpr) nodes.PhysicalPlanNode {
	col, val := extractEquality(pred)
	if col == nil {
		return nil
	}
	idx := o.findIndex(table, col, types.IndexTypeHash)
	if idx == nil {
		return nil
	}
	return nodes.NewHashIndexScanNode(table, idx, val)
}

func (o *Optimizer) tryBTreeScan(table *model.TableDefinition, pred semantic.ResolvedExpr) nodes.PhysicalPlanNode {
	rng := o.extractRange(pred)
	if rng == nil {
		return nil
	}
	idx := o.findIndex(table, rng.column, types.IndexTypeBTree)
	if idx == nil {
		return nil
	}
	return nodes.NewBTreeIndexScanNode(table, idx, rng.from, rng.fromInclusive, rng.to, rng.toInclusive)
}

func (o *Optimizer) findIndex(table *model.TableDefinition, col *model.ColumnDefinition, t types.IndexType) *model.IndexDefinition {
	indexes, _ := o.catalog.ListIndexes(table)
	for _, idx := range indexes {
		if idx.IndexType() == t && idx.ColumnOid() == col.Oid() {
			return idx
		}
	}
	return nil
}

func (o *Optimizer) extractRange(pred semantic.ResolvedExpr) *rangeInfo {
	b, ok := pred.(*semantic.ResolvedBinaryExpr)
	if !ok {
		return nil
	}
	if b.Op() == "AND" {
		return o.mergeAndRanges(b.Left(), b.Right())
	}
	return o.extractSingleRange(b)
}

func (o *Optimizer) mergeAndRanges(left, right semantic.ResolvedExpr) *rangeInfo {
	l, r := o.extractRange(left), o.extractRange(right)
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

func (o *Optimizer) extractSingleRange(b *semantic.ResolvedBinaryExpr) *rangeInfo {
	col, val, op := o.extractColValOp(b)
	if col == nil {
		return nil
	}
	builder, ok := o.rangeBuilders[op]
	if !ok {
		return nil
	}
	r := &rangeInfo{column: col}
	builder(r, val)
	return r
}

func (o *Optimizer) extractColValOp(b *semantic.ResolvedBinaryExpr) (*model.ColumnDefinition, any, string) {
	if col, val, ok := tryExtractColVal(b.Left(), b.Right()); ok {
		return col, val, b.Op()
	}
	if col, val, ok := tryExtractColVal(b.Right(), b.Left()); ok {
		return col, val, o.invertOp(b.Op())
	}
	return nil, nil, ""
}

func (o *Optimizer) invertOp(op string) string {
	if inv, ok := o.invertedOps[op]; ok {
		return inv
	}
	return op
}
