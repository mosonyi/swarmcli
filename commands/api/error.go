// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api //nolint:revive // standard short package name

import "fmt"

var ErrEmptyCommand = fmt.Errorf("empty command")

func ErrUnknownCommand(input string) error {
	return fmt.Errorf("unknown command: %s", input)
}
