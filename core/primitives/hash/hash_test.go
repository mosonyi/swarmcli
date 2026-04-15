// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package hash //nolint:revive // matches stdlib name intentionally

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFmt_Zero(t *testing.T) {
	require.Equal(t, "0000000000000000", Fmt(0))
}

func TestFmt_MaxUint64(t *testing.T) {
	require.Equal(t, "ffffffffffffffff", Fmt(math.MaxUint64))
}

func TestFmt_KnownValues(t *testing.T) {
	require.Equal(t, "00000000000000ff", Fmt(255))
	require.Equal(t, "0000000000000100", Fmt(256))
}

func TestCompute_Struct(t *testing.T) {
	type sample struct {
		Name string
		Age  int
	}
	h1, err := Compute(sample{Name: "alice", Age: 30})
	require.NoError(t, err)

	h2, err := Compute(sample{Name: "alice", Age: 30})
	require.NoError(t, err)

	require.Equal(t, h1, h2, "same struct should produce same hash")
}

func TestCompute_DifferentStructs(t *testing.T) {
	type sample struct {
		Name string
		Age  int
	}
	h1, err := Compute(sample{Name: "alice", Age: 30})
	require.NoError(t, err)

	h2, err := Compute(sample{Name: "bob", Age: 25})
	require.NoError(t, err)

	require.NotEqual(t, h1, h2, "different structs should produce different hashes")
}

func TestCompute_Nil(t *testing.T) {
	h, err := Compute(nil)
	require.NoError(t, err)
	require.NotEmpty(t, Fmt(h))
}
