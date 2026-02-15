package planner

import (
	"reflect"

	"GopherDB/internal/core/planner/nodes"
	"GopherDB/internal/core/sql/semantic"
)

type queryPlanner func(semantic.QueryTree) nodes.LogicalPlanNode

type DefaultPlanner struct {
	planners map[reflect.Type]queryPlanner
}

func NewDefaultPlanner() *DefaultPlanner {
	planner := &DefaultPlanner{}
	planner.planners = map[reflect.Type]queryPlanner{
		reflect.TypeOf((*semantic.CreateTableQueryTree)(nil)): planner.planCreateTable,
		reflect.TypeOf((*semantic.InsertQueryTree)(nil)):      planner.planInsert,
		reflect.TypeOf((*semantic.CreateIndexQueryTree)(nil)): planner.planCreateIndex,
		reflect.TypeOf((*semantic.SelectQueryTree)(nil)):      planner.planSelect,
		reflect.TypeOf((*semantic.ExplainQueryTree)(nil)):     planner.planExplain,
	}
	return planner
}

func (planner *DefaultPlanner) Plan(query semantic.QueryTree) nodes.LogicalPlanNode {
	if fn, ok := planner.planners[reflect.TypeOf(query)]; ok {
		return fn(query)
	}
	return nil
}

func (planner *DefaultPlanner) planCreateTable(query semantic.QueryTree) nodes.LogicalPlanNode {
	return nodes.NewCreateTableNode(query.(*semantic.CreateTableQueryTree))
}

func (planner *DefaultPlanner) planInsert(query semantic.QueryTree) nodes.LogicalPlanNode {
	return nodes.NewInsertNode(query.(*semantic.InsertQueryTree))
}

func (planner *DefaultPlanner) planCreateIndex(query semantic.QueryTree) nodes.LogicalPlanNode {
	return nodes.NewCreateIndexNode(query.(*semantic.CreateIndexQueryTree))
}

func (planner *DefaultPlanner) planSelect(query semantic.QueryTree) nodes.LogicalPlanNode {
	sel := query.(*semantic.SelectQueryTree)
	var node nodes.LogicalPlanNode = nodes.NewScanNode(sel.Table())
	if sel.Filter() != nil {
		node = nodes.NewFilterNode(node, sel.Filter())
	}
	node = nodes.NewProjectNode(node, sel.TargetColumns())
	return node
}

func (planner *DefaultPlanner) planExplain(query semantic.QueryTree) nodes.LogicalPlanNode {
	explain := query.(*semantic.ExplainQueryTree)
	return nodes.NewExplainNode(planner.Plan(explain.Inner()))
}
