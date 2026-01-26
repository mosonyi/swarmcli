// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api

import "fmt"

var ErrEmptyCommand = fmt.Errorf("empty command")

func ErrUnknownCommand(input string) error {
	return fmt.Errorf("unknown command: %s", input)
}
