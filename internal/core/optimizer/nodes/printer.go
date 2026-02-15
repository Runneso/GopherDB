package nodes

import "strings"

func PrintPlan(node PhysicalPlanNode) string {
	var builder strings.Builder
	printNode(&builder, node, 0)
	return builder.String()
}

func printNode(builder *strings.Builder, node PhysicalPlanNode, indent int) {
	for index := 0; index < indent; index++ {
		builder.WriteString("  ")
	}
	builder.WriteString(node.DisplayName())
	builder.WriteString("\n")
	for _, child := range node.Children() {
		printNode(builder, child, indent+1)
	}
}
