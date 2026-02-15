package planner

import (
	"reflect"

	"GopherDB/internal/core/planner/nodes"
	"GopherDB/internal/core/sql/semantic"
)

type queryPlanner func(semantic.QueryTree) nodes.LogicalPlanNode

type Planner struct {
	planners map[reflect.Type]queryPlanner
}

func NewPlanner() *Planner {
	planner := &Planner{}
	planner.planners = map[reflect.Type]queryPlanner{
		reflect.TypeOf((*semantic.CreateTableQueryTree)(nil)): planner.planCreateTable,
		reflect.TypeOf((*semantic.InsertQueryTree)(nil)):      planner.planInsert,
		reflect.TypeOf((*semantic.CreateIndexQueryTree)(nil)): planner.planCreateIndex,
		reflect.TypeOf((*semantic.SelectQueryTree)(nil)):      planner.planSelect,
		reflect.TypeOf((*semantic.ExplainQueryTree)(nil)):     planner.planExplain,
	}
	return planner
}

func (planner *Planner) Plan(query semantic.QueryTree) nodes.LogicalPlanNode {
	if planner, ok := planner.planners[reflect.TypeOf(query)]; ok {
		return planner(query)
	}
	return nil
}

func (planner *Planner) planCreateTable(query semantic.QueryTree) nodes.LogicalPlanNode {
	return nodes.NewCreateTableNode(query.(*semantic.CreateTableQueryTree))
}

func (planner *Planner) planInsert(query semantic.QueryTree) nodes.LogicalPlanNode {
	return nodes.NewInsertNode(query.(*semantic.InsertQueryTree))
}

func (planner *Planner) planCreateIndex(query semantic.QueryTree) nodes.LogicalPlanNode {
	return nodes.NewCreateIndexNode(query.(*semantic.CreateIndexQueryTree))
}

func (planner *Planner) planSelect(query semantic.QueryTree) nodes.LogicalPlanNode {
	sel := query.(*semantic.SelectQueryTree)
	var node nodes.LogicalPlanNode = nodes.NewScanNode(sel.Table())
	if sel.Filter() != nil {
		node = nodes.NewFilterNode(node, sel.Filter())
	}
	node = nodes.NewProjectNode(node, sel.TargetColumns())
	return node
}

func (planner *Planner) planExplain(query semantic.QueryTree) nodes.LogicalPlanNode {
	explain := query.(*semantic.ExplainQueryTree)
	return nodes.NewExplainNode(planner.Plan(explain.Inner()))
}
