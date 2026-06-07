// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartialListError_Message(t *testing.T) {
	require.Equal(t, "1 node unreachable", (&PartialListError{
		NodeErrors: map[string]string{"n1": "timeout"},
	}).Error())
	require.Equal(t, "2 nodes unreachable", (&PartialListError{
		NodeErrors: map[string]string{"n1": "timeout", "n2": "refused"},
	}).Error())
	// Note overrides the node-count message.
	require.Equal(t, "connected node only", (&PartialListError{
		Note:       "connected node only",
		NodeErrors: map[string]string{"n1": "timeout"},
	}).Error())
}

func TestPartialListError_RoundTripsThroughErrorsAs(t *testing.T) {
	orig := &PartialListError{NodeErrors: map[string]string{"n1": "timeout"}}
	// Wrapped, as a caller might return it.
	wrapped := fmt.Errorf("listing failed: %w", orig)

	var got *PartialListError
	require.True(t, errors.As(wrapped, &got))
	require.Equal(t, orig.NodeErrors, got.NodeErrors)
}
