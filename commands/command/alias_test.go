// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAliasCommand_AliasOf(t *testing.T) {
	a := aliasCommand{name: "ctx", target: contextsCmd}
	require.Equal(t, contextsCmd.Name(), a.AliasOf())
}
