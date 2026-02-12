// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import "github.com/docker/docker/client"

// ClientOps abstracts Docker client lifecycle for testability and extensibility.
type ClientOps interface {
	GetClient() (*client.Client, error)
	ResetClient()
}

type defaultClientOps struct{}

func (defaultClientOps) GetClient() (*client.Client, error) { return GetClient() }
func (defaultClientOps) ResetClient()                       { ResetClient() }
