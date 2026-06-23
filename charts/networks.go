// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// externalNetworks returns the real names of the networks a rendered stack
// manifest declares as external. Non-external networks are ignored; a manifest
// without a networks block (or invalid YAML) yields no names and no error —
// deploy validation handles that.
func externalNetworks(manifest string) ([]string, error) {
	return externalResourceNames(manifest, "networks"), nil
}

// externalResourceNames returns the real names of the entries under the given
// top-level compose key (e.g. "networks", "secrets", "configs") that the
// manifest declares external. It understands both compose forms:
//
//	<key>:
//	  alias:
//	    external: true            # real name == map key
//	  other:
//	    external:
//	      name: actual-resource   # real name == external.name
//
// Non-external entries are ignored. A manifest without that block, or invalid
// YAML, yields no names. The block is decoded in isolation so unrelated
// top-level keys (services, volumes, …) of any shape do not interfere.
func externalResourceNames(manifest, topLevelKey string) []string {
	var top map[string]yaml.Node
	if err := yaml.Unmarshal([]byte(manifest), &top); err != nil {
		return nil
	}
	node, ok := top[topLevelKey]
	if !ok {
		return nil
	}
	var block map[string]struct {
		External yaml.Node `yaml:"external"`
	}
	if err := node.Decode(&block); err != nil {
		return nil
	}
	var names []string
	for key, res := range block {
		switch res.External.Kind {
		case yaml.ScalarNode:
			var b bool
			if err := res.External.Decode(&b); err == nil && b {
				names = append(names, key)
			}
		case yaml.MappingNode:
			var m struct {
				Name string `yaml:"name"`
			}
			if err := res.External.Decode(&m); err == nil {
				if m.Name != "" {
					names = append(names, m.Name)
				} else {
					names = append(names, key)
				}
			}
		}
	}
	sort.Strings(names)
	return names
}
