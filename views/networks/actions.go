// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package networksview

import (
	"fmt"
	"swarmcli/docker"
	"time"

	"github.com/docker/docker/api/types/network"
)

type networkItem struct {
	Name      string
	ID        string
	Driver    string
	Scope     string
	CreatedAt time.Time
	Ingress   bool // true if this is the swarm routing-mesh ingress network
	Used      bool // true if used by any service
	UsedKnown bool // true if Used has been computed (false => loading/unknown)
}

func (i networkItem) FilterValue() string { return i.Name }
func (i networkItem) Title() string       { return i.Name }
func (i networkItem) Description() string {
	createdStr := "N/A"
	if !i.CreatedAt.IsZero() {
		createdStr = i.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("ID: %s        Driver: %s        Scope: %s        Created: %s",
		i.ID, i.Driver, i.Scope, createdStr)
}

type usedByItem struct {
	StackName   string
	ServiceName string
}

func (i usedByItem) FilterValue() string { return i.StackName + " " + i.ServiceName }
func (i usedByItem) Title() string       { return fmt.Sprintf("%-24s %-24s", i.StackName, i.ServiceName) }
func (i usedByItem) Description() string { return "Service: " + i.ServiceName }

type networkWithUsage struct {
	Network  network.Summary
	Services []string
}

func (nw *networkWithUsage) PrettyJSON() ([]byte, error) {
	dockerNW := docker.NetworkWithUsage{
		Network:  nw.Network,
		Services: nw.Services,
	}
	return dockerNW.PrettyJSON()
}

