package inspectview

import "strings"

// markMatches sets Matches true for any node whose key or value contains term (case-insensitive)
// and clears previous Matches.
func (m *Model) markMatches(term string) {
	term = strings.ToLower(term)
	var walk func(n *Node) bool
	walk = func(n *Node) bool {
		n.Matches = false
		hit := false
		// check key
		if strings.Contains(strings.ToLower(n.Key), term) {
			hit = true
		}
		// check value
		if n.ValueStr != "" && strings.Contains(strings.ToLower(n.ValueStr), term) {
			hit = true
		} else if n.ValueStr == "" {
			// for non-leaf nodes, we might want to check textual representation of subtree keys/values
			// but keep it simple: check child keys recursively
			for _, c := range n.Children {
				if walk(c) {
					hit = true
				}
			}
			// Note: calling walk(c) twice avoided by logic: but here we need children results.
			// To avoid double traversal, implement second pass below:
		}
		n.Matches = hit
		return hit
	}
	// Because the above walk both checks children recursively and sets Matches,
	// run a simpler approach: first clear Matches, then mark nodes whose key/value match; then propagate matching flags to ancestors.
	var mark func(n *Node)
	mark = func(n *Node) {
		n.Matches = false
		if strings.Contains(strings.ToLower(n.Key), term) {
			n.Matches = true
		}
		if n.ValueStr != "" && strings.Contains(strings.ToLower(n.ValueStr), term) {
			n.Matches = true
		}
		for _, c := range n.Children {
			mark(c)
			if c.Matches {
				n.Matches = true
			}
		}
	}
	mark(m.Root)
	// expand ancestors of matches so they become visible
	var expandAncestors func(n *Node)
	expandAncestors = func(n *Node) {
		if n == nil {
			return
		}
		if n.Parent != nil {
			n.Parent.Expanded = true
			expandAncestors(n.Parent)
		}
	}
	var walkAndExpand func(n *Node)
	walkAndExpand = func(n *Node) {
		if n.Matches {
			expandAncestors(n)
		}
		for _, c := range n.Children {
			walkAndExpand(c)
		}
	}
	walkAndExpand(m.Root)
}
