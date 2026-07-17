// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// parseRequirements parses and validates a chart's requirements.yaml, applying
// defaults so the rest of the engine can rely on every field being populated:
// each network's Driver defaults to "overlay" and its AutoCreate/Attachable
// default to true. Every entry must carry a non-empty name.
func parseRequirements(data []byte) (*Requirements, error) {
	req, err := unmarshalRequirements(data)
	if err != nil {
		return nil, err
	}
	if err := validateRequirements(req); err != nil {
		return nil, err
	}
	return req, nil
}

// unmarshalRequirements decodes requirements.yaml WITHOUT validating it. Chart
// loading needs the two steps apart: requirements.yaml is a Go template, so a chart
// may `range` over a user-supplied list and the raw bytes then do not parse as YAML
// at all — which is not an error. A file that DOES parse but declares something
// invalid still is.
func unmarshalRequirements(data []byte) (*Requirements, error) {
	var req Requirements
	if err := yaml.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse %s: %w", requirementsName, err)
	}
	return &req, nil
}

func validateRequirements(req *Requirements) error {
	for i := range req.Networks {
		n := &req.Networks[i]
		if n.Name == "" {
			return fmt.Errorf("%s: networks[%d] has no name", requirementsName, i)
		}
		if n.Driver == "" {
			n.Driver = "overlay"
		}
		if n.Attachable == nil {
			n.Attachable = boolPtr(true)
		}
		if n.AutoCreate == nil {
			n.AutoCreate = boolPtr(true)
		}
	}
	for i, s := range req.Secrets {
		if s.Name == "" {
			return fmt.Errorf("%s: secrets[%d] has no name", requirementsName, i)
		}
	}
	for i, c := range req.Configs {
		if c.Name == "" {
			return fmt.Errorf("%s: configs[%d] has no name", requirementsName, i)
		}
	}
	return nil
}

func boolPtr(b bool) *bool { return &b }

// network returns the declared requirement for the network of the given real
// name, if any. After parseRequirements its bool fields are non-nil.
func (r *Requirements) network(name string) (*NetworkRequirement, bool) {
	if r == nil {
		return nil, false
	}
	for i := range r.Networks {
		if r.Networks[i].Name == name {
			return &r.Networks[i], true
		}
	}
	return nil, false
}

// secret returns the declared requirement for the secret of the given real
// name, if any.
func (r *Requirements) secret(name string) (*ResourceRequirement, bool) {
	if r == nil {
		return nil, false
	}
	for i := range r.Secrets {
		if r.Secrets[i].Name == name {
			return &r.Secrets[i], true
		}
	}
	return nil, false
}

// config returns the declared requirement for the config of the given real
// name, if any.
func (r *Requirements) config(name string) (*ResourceRequirement, bool) {
	if r == nil {
		return nil, false
	}
	for i := range r.Configs {
		if r.Configs[i].Name == name {
			return &r.Configs[i], true
		}
	}
	return nil, false
}
