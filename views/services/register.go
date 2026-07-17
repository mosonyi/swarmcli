// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(deps docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	v := New(w, h)
	v.deps = deps

	data, _ := payload.(map[string]any)

	filterType := AllFilter
	var nodeID, stackName string
	var serviceName string
	var selectServiceName string
	var noStack bool

	if n, ok := data["nodeID"].(string); ok {
		filterType = NodeFilter
		nodeID = n
	}
	if s, ok := data["stackName"].(string); ok {
		filterType = StackFilter
		stackName = s
	}
	if b, ok := data["noStack"].(bool); ok {
		noStack = b
	}
	if noStack {
		filterType = NoStackFilter
		stackName = ""
		nodeID = ""
	}
	if s, ok := data["serviceName"].(string); ok {
		serviceName = s
	}
	if s, ok := data["selectServiceName"].(string); ok {
		selectServiceName = s
	}
	if serviceName != "" {
		v.List.Query = serviceName
	}
	if selectServiceName != "" {
		v.SetPendingSelectServiceName(selectServiceName)
	}

	entries, scope := v.loadServicesForView(filterType, nodeID, stackName)

	// Apply data directly so keys work immediately (no race with async Cmd).
	msg := Msg{
		Scope:      scope,
		Entries:    entries,
		FilterType: filterType,
		NodeID:     nodeID,
		StackName:  stackName,
	}
	v.lastSnapshot, _ = hash.Compute(entries)
	v.SetContent(msg)
	v.Visible = true

	return v, tickCmd()
}
