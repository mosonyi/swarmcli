package inspectview

import (
	"encoding/json"
	"fmt"
)

func ParseJSON(jsonStr string) (*treeNode, error) {
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}
	root := &treeNode{
		Key:   "root",
		Value: data,
		Path:  "root",
	}
	buildChildren(root)
	return root, nil
}

func buildChildren(node *treeNode) {
	switch v := node.Value.(type) {
	case map[string]any:
		for k, val := range v {
			child := &treeNode{
				Key:   k,
				Value: val,
				Path:  node.Path + "." + k,
			}
			buildChildren(child)
			node.Children = append(node.Children, child)
		}
	case []any:
		for i, val := range v {
			child := &treeNode{
				Key:   fmt.Sprintf("[%d]", i),
				Value: val,
				Path:  fmt.Sprintf("%s[%d]", node.Path, i),
			}
			buildChildren(child)
			node.Children = append(node.Children, child)
		}
	}
}
