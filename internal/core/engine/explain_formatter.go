package engine

import (
	"fmt"
	"strings"

	"GopherDB/internal/core/optimizer/nodes"
	plannernodes "GopherDB/internal/core/planner/nodes"
	"GopherDB/internal/core/sql/ast"
	"GopherDB/internal/core/sql/lexer"
	"GopherDB/internal/core/sql/semantic"
)

func FormatExplain(tokens []*lexer.Token, statement ast.Statement, queryTree semantic.QueryTree, logicalPlan plannernodes.LogicalPlanNode, physicalPlan nodes.PhysicalPlanNode) string {
	var builder strings.Builder

	builder.WriteString("TOKENS:\n")
	for _, t := range tokens {
		builder.WriteString("  ")
		builder.WriteString(fmt.Sprintf("%s(%s)", t.Type().String(), t.Text()))
		builder.WriteString("\n")
	}

	builder.WriteString("\nAST:\n")
	builder.WriteString(fmt.Sprintf("  %T\n", statement))

	builder.WriteString("\nQUERY_TREE:\n")
	builder.WriteString(fmt.Sprintf("  %s\n", queryTree.Type().String()))

	builder.WriteString("\nLOGICAL_PLAN:\n")
	builder.WriteString(plannernodes.PrintPlan(logicalPlan))

	builder.WriteString("\nPHYSICAL_PLAN:\n")
	builder.WriteString(nodes.PrintPlan(physicalPlan))

	return builder.String()
}
