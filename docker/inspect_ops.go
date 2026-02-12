// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import "context"

// InspectOps abstracts resource inspection for testability and extensibility.
type InspectOps interface {
	Inspect(ctx context.Context, t InspectType, id string) (string, error)
}

type defaultInspectOps struct{}

func (defaultInspectOps) Inspect(ctx context.Context, t InspectType, id string) (string, error) {
	return Inspect(ctx, t, id)
}
