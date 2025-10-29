package inspectview

import (
	"fmt"
	"strings"
)

func RenderTree(node *treeNode, indent int, expanded map[string]bool) []string {
	lines := []string{}
	prefix := strings.Repeat("  ", indent)
	symbol := "-"
	if len(node.Children) > 0 {
		if expanded[node.Path] {
			symbol = "▼"
		} else {
			symbol = "▶"
		}
	}

	switch node.Value.(type) {
	case map[string]any, []any:
		lines = append(lines, fmt.Sprintf("%s%s %s", prefix, symbol, node.Key))
	default:
		lines = append(lines, fmt.Sprintf("%s%s: %v", prefix, node.Key, node.Value))
	}

	if len(node.Children) > 0 && expanded[node.Path] {
		for _, child := range node.Children {
			childLines := RenderTree(child, indent+1, expanded)
			lines = append(lines, childLines...)
		}
	}
	return lines
}
