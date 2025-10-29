package inspectview

// lineToPath returns the path of the node at the given line index in the rendered tree.
// It recursively traverses the tree, counting lines as it goes.
func lineToPath(node *treeNode, targetLine int, currentLine int, expanded map[string]bool) string {
	// Count this node as the current line
	if currentLine == targetLine {
		return node.Path
	}
	currentLine++

	// Only traverse children if expanded
	if len(node.Children) > 0 && expanded[node.Path] {
		for _, child := range node.Children {
			linesConsumed := countLines(child, expanded)
			if targetLine < currentLine+linesConsumed {
				// The target line is inside this child
				return lineToPath(child, targetLine, currentLine, expanded)
			}
			currentLine += linesConsumed
		}
	}

	return "" // not found
}

// countLines returns how many lines this node would render
func countLines(node *treeNode, expanded map[string]bool) int {
	lines := 1 // current node
	if len(node.Children) > 0 && expanded[node.Path] {
		for _, child := range node.Children {
			lines += countLines(child, expanded)
		}
	}
	return lines
}
