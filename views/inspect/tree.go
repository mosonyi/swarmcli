package inspectview

import (
	"encoding/json"
	"fmt"
)

func ParseJSON(jsonStr string) (*Node, error) {
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}
	root := &Node{
		Key:      "root",
		Raw:      data,
		Expanded: true,
		Path:     "root",
	}
	buildChildren(root, 0)
	return root, nil
}

func buildChildren(n *Node, depth int) {
	n.Depth = depth
	switch v := n.Raw.(type) {
	case map[string]any:
		// preserve stable order by iterating keys sorted? For now iterate as-is.
		for k, val := range v {
			child := &Node{
				Key:      k,
				Raw:      val,
				Parent:   n,
				Expanded: false,
				Path:     n.Path + "." + k,
			}
			buildChildren(child, depth+1)
			n.Children = append(n.Children, child)
		}
	case []any:
		for i, val := range v {
			child := &Node{
				Key:      fmt.Sprintf("[%d]", i),
				Raw:      val,
				Parent:   n,
				Expanded: false,
				Path:     fmt.Sprintf("%s[%d]", n.Path, i),
			}
			buildChildren(child, depth+1)
			n.Children = append(n.Children, child)
		}
	default:
		n.ValueStr = fmt.Sprintf("%v", v)
	}
}

// ---------------------- Visible list & rendering ----------------------

// rebuildVisible updates m.Visible based on expansion state and search
func (m *Model) rebuildVisible() {
	if m.Root == nil {
		m.Visible = nil
		return
	}
	if m.SearchTerm == "" {
		// produce linear list from expanded tree
		var list []*Node
		m.collectVisible(m.Root, &list)
		m.Visible = list
	} else {
		// filter: matches and ancestors only
		m.markMatches(m.SearchTerm)
		var list []*Node
		m.collectFiltered(m.Root, &list)
		m.Visible = list
		// set searchIndex to first match if any
		m.searchIndex = -1
		for i, n := range m.Visible {
			if n.Matches {
				m.searchIndex = i
				break
			}
		}
		if m.searchIndex >= 0 {
			m.Cursor = m.searchIndex
		} else {
			m.Cursor = 0
		}
	}
}

// collectVisible appends nodes following expanded flags
func (m *Model) collectVisible(n *Node, out *[]*Node) {
	// always include the node itself
	*out = append(*out, n)
	if len(n.Children) == 0 {
		return
	}
	if n.Expanded {
		for _, c := range n.Children {
			m.collectVisible(c, out)
		}
	}
}

// collectFiltered adds nodes that are matches or ancestors of matches (and their visible children)
func (m *Model) collectFiltered(n *Node, out *[]*Node) {
	if n == nil {
		return
	}
	// only include if this node is match or ancestor of match
	if !n.Matches {
		// check if any descendant matches
		found := false
		var walk func(x *Node)
		walk = func(x *Node) {
			for _, c := range x.Children {
				if c.Matches {
					found = true
					return
				}
				walk(c)
				if found {
					return
				}
			}
		}
		walk(n)
		if !found {
			return
		}
	}
	// include this node
	*out = append(*out, n)
	// include visible children if expanded (or if forced by matching)
	for _, c := range n.Children {
		m.collectFiltered(c, out)
	}
}
