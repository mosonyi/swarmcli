// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api //nolint:revive // standard short package name

import (
	"fmt"

	"github.com/Eldara-Tech/swarmcli/v2/registry"
)

var ErrEmptyCommand = fmt.Errorf("empty command")

func ErrUnknownCommand(input string) error {
	return fmt.Errorf("unknown command: %s", input)
}

// ErrHelpRequested is a sentinel returned by ParseInput when the user
// asked for a command's help (`:cmd --help` / `-h`). It is not a parse
// failure: the dispatcher recovers it via errors.As and renders the
// per-command help screen instead of an inline error.
type ErrHelpRequested struct {
	Cmd registry.Command
}

func (ErrHelpRequested) Error() string { return "help requested" }
