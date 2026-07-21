// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

// hostFor builds a client from the options and reports the host the SDK would
// actually address. Constructing a client does not dial, so this runs without a
// daemon; it is the closest observable proxy for "which transport was chosen".
func hostFor(t *testing.T, host, ca, cert, key string, skipVerify bool) string {
	t.Helper()
	opts, err := clientOptsFor(host, ca, cert, key, skipVerify)
	require.NoError(t, err)
	cli, err := client.NewClientWithOpts(opts...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })
	return cli.DaemonHost()
}

// An ssh:// endpoint is rewritten to the connection helper's dummy host, which
// is the observable half of the fix: the SDK never sees "ssh://" and so never
// hands it to go-connections, which special-cases only unix and npipe and would
// TCP-dial it. The dialer that replaces it runs `ssh … docker system dial-stdio`.
func TestSSHHostGoesThroughAConnectionHelper(t *testing.T) {
	got := hostFor(t, "ssh://user@manager.example.com", "", "", "", false)
	require.NotEqual(t, "ssh://user@manager.example.com", got)
	require.NotEmpty(t, got)
}

// Every endpoint the SDK can dial itself is passed through untouched. This is
// the regression guard that matters: the connection-helper lookup runs on every
// context, so it must be inert for the transports that already worked.
func TestNonHelperHostsArePassedThroughUnchanged(t *testing.T) {
	// npipe:// is deliberately absent: the SDK refuses to construct a client for
	// it off Windows ("protocol not available"), which says nothing about this
	// change. unix:// and tcp:// are the transports that actually run here.
	for _, host := range []string{
		"unix:///var/run/docker.sock",
		"tcp://10.0.0.5:2376",
	} {
		require.Equal(t, host, hostFor(t, host, "", "", "", false), "host %q", host)
	}
}

// TLS material is applied only on the direct path. An ssh context is secured by
// ssh itself, and certs sitting in its storage describe a tcp:// endpoint it is
// not using — layering them over the dial-stdio pipe would break the connection
// rather than protect it.
func TestConnectionHelperIgnoresTLSMaterial(t *testing.T) {
	dir := t.TempDir()
	ca := filepath.Join(dir, "ca.pem")
	cert := filepath.Join(dir, "cert.pem")
	key := filepath.Join(dir, "key.pem")
	for _, p := range []string{ca, cert, key} {
		require.NoError(t, os.WriteFile(p, []byte("not a certificate"), 0o600))
	}

	// With a helper the certs are ignored, so building the client succeeds even
	// though the files are not valid PEM. On the direct path the same files are
	// read and rejected, which proves they would have been used.
	require.NotEmpty(t, hostFor(t, "ssh://user@manager.example.com", ca, cert, key, false))

	opts, err := clientOptsFor("tcp://10.0.0.5:2376", ca, cert, key, false)
	require.NoError(t, err)
	_, err = client.NewClientWithOpts(opts...)
	require.Error(t, err)
}

// A malformed ssh URL is reported rather than silently falling back to a
// TCP dial that would fail later with an unrelated message.
func TestMalformedSSHHostIsReported(t *testing.T) {
	_, err := clientOptsFor("ssh://", "", "", "", false)
	require.ErrorContains(t, err, "connection helper")
}
