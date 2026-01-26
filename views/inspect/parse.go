// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

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
		Key: "root",
		Raw: data,
	}
	buildChildren(root, data)
	return root, nil
}

func buildChildren(parent *Node, data any) {
	switch v := data.(type) {
	case map[string]any:
		for k, val := range v {
			child := &Node{
				Key:    k,
				Raw:    val,
				Parent: parent,
			}
			buildChildren(child, val)
			parent.Children = append(parent.Children, child)
		}
	case []any:
		for i, val := range v {
			child := &Node{
				Key:    fmt.Sprintf("[%d]", i),
				Raw:    val,
				Parent: parent,
			}
			buildChildren(child, val)
			parent.Children = append(parent.Children, child)
		}
	default:
		parent.ValueStr = fmt.Sprintf("%v", v)
	}
}
