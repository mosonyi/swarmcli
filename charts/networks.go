// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// externalNetworks returns the real names of the networks a rendered stack
// manifest declares as external. Non-external networks are ignored; a manifest
// without a networks block (or invalid YAML) yields no names and no error —
// deploy validation handles that.
func externalNetworks(manifest string) ([]string, error) {
	return externalResourceNames(manifest, "networks")
}

// externalResourceNames returns the real names of the entries under the given
// top-level compose key (e.g. "networks", "secrets", "configs") that the
// manifest declares external. It understands both compose forms:
//
//	<key>:
//	  alias:
//	    external: true
//	    name: actual-resource   # real name == name, else the map key
//	  other:
//	    external:
//	      name: actual-resource # deprecated form; real name == external.name
//
// Precedence follows docker/cli's compose loader, which is what
// `docker stack deploy` runs against the very same manifest: a sibling name:
// wins, the deprecated external.name is still honoured, and the map key is only
// the fallback. Reading the key when a name: was given is what made a chart
// written to the current spec impossible to deploy (#513) — the pre-flight
// looked for something nothing was ever going to be called.
//
// Declaring both forms is an error in the loader, so it is one here too:
// accepting it would let a manifest clear the pre-flight that the deploy then
// rejects, which is the opposite of what a pre-flight is for.
//
// Non-external entries are ignored. A manifest without that block, or invalid
// YAML, yields no names. The block is decoded in isolation so unrelated
// top-level keys (services, volumes, …) of any shape do not interfere.
func externalResourceNames(manifest, topLevelKey string) ([]string, error) {
	var top map[string]yaml.Node
	if err := yaml.Unmarshal([]byte(manifest), &top); err != nil {
		return nil, nil
	}
	node, ok := top[topLevelKey]
	if !ok {
		return nil, nil
	}
	var block map[string]struct {
		External yaml.Node `yaml:"external"`
		Name     string    `yaml:"name"`
	}
	if err := node.Decode(&block); err != nil {
		return nil, nil
	}
	var names []string
	for key, res := range block {
		// deprecated holds external.name, set only by the long form.
		var deprecated string
		switch res.External.Kind {
		case yaml.ScalarNode:
			var b bool
			if err := res.External.Decode(&b); err != nil || !b {
				continue
			}
		case yaml.MappingNode:
			var m struct {
				Name string `yaml:"name"`
			}
			if err := res.External.Decode(&m); err != nil {
				continue
			}
			deprecated = m.Name
		default:
			continue
		}
		switch {
		case res.Name != "" && deprecated != "":
			return nil, fmt.Errorf("%s '%s': external.name and name conflict; use only name",
				strings.TrimSuffix(topLevelKey, "s"), key)
		case res.Name != "":
			names = append(names, res.Name)
		case deprecated != "":
			names = append(names, deprecated)
		default:
			names = append(names, key)
		}
	}
	sort.Strings(names)
	return names, nil
}
