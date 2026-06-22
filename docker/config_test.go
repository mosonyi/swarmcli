// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigDisplayData(t *testing.T) {
	// gzip payloads (e.g. chart release records) are transparently decompressed.
	plain := []byte("name: whoami\nrevision: 1\n")
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(plain)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	gz := &ConfigWithDecodedData{Data: buf.Bytes()}
	require.Equal(t, plain, gz.DisplayData())

	// non-gzip payloads are returned unchanged.
	raw := &ConfigWithDecodedData{Data: []byte("plain config text")}
	require.Equal(t, []byte("plain config text"), raw.DisplayData())
}
