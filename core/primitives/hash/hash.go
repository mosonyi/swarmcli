// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package hash //nolint:revive // matches stdlib name intentionally

import (
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
)

func Fmt(h uint64) string {
	return fmt.Sprintf("%016x", h)
}

func Compute(v any) (uint64, error) {
	return hashstructure.Hash(v, hashstructure.FormatV2, nil)
}
