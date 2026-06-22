// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// externalNetworks returns the real names of the networks a rendered stack
// manifest declares as external. It understands both compose forms:
//
//	networks:
//	  traefik-public:
//	    external: true            # real name == map key
//	  alias:
//	    external:
//	      name: actual-network    # real name == external.name
//
// Non-external networks are ignored. A manifest without a networks block (or
// invalid YAML) yields no names and no error — deploy validation handles that.
func externalNetworks(manifest string) ([]string, error) {
	var doc struct {
		Networks map[string]struct {
			External yaml.Node `yaml:"external"`
		} `yaml:"networks"`
	}
	if err := yaml.Unmarshal([]byte(manifest), &doc); err != nil {
		return nil, nil
	}
	var names []string
	for key, net := range doc.Networks {
		switch net.External.Kind {
		case yaml.ScalarNode:
			var b bool
			if err := net.External.Decode(&b); err == nil && b {
				names = append(names, key)
			}
		case yaml.MappingNode:
			var m struct {
				Name string `yaml:"name"`
			}
			if err := net.External.Decode(&m); err == nil {
				if m.Name != "" {
					names = append(names, m.Name)
				} else {
					names = append(names, key)
				}
			}
		}
	}
	sort.Strings(names)
	return names, nil
}
