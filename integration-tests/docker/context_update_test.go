// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package docker

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Eldara-Tech/swarmcli/docker"
)

// The context these tests create and remove. Every case works on its own
// throwaway context, never on the one the suite runs against.
const probeContext = "swarmcli-context-update-probe"

// TestUpdateContextEndpoint_PreservesTLSMaterial is the regression guard for
// the reason this update is not a bare `--docker host=…`: Docker replaces a
// context's whole endpoint and resets its TLS material to exactly what the
// command names, so omitting the certificates deletes them.
func TestUpdateContextEndpoint_PreservesTLSMaterial(t *testing.T) {
	requireDocker(t)
	certs := writeTLSMaterial(t)
	createProbeContext(t, "tcp://127.0.0.1:2376", certs)

	if err := docker.UpdateContextEndpoint(probeContext, "", "tcp://10.0.0.7:2376"); err != nil {
		t.Fatalf("UpdateContextEndpoint failed: %v", err)
	}

	endpoint, err := docker.InspectContextEndpoint(probeContext)
	if err != nil {
		t.Fatalf("InspectContextEndpoint failed: %v", err)
	}
	if endpoint.Host != "tcp://10.0.0.7:2376" {
		t.Errorf("host = %q, want the updated endpoint", endpoint.Host)
	}
	if !endpoint.HasTLS() {
		t.Fatalf("the update deleted the TLS material: %+v", endpoint)
	}
	// The paths are the store's own copies, so compare the bytes rather than
	// trusting that the files are merely present.
	for name, stored := range map[string]string{
		"ca.pem":   endpoint.CAFile,
		"cert.pem": endpoint.CertFile,
		"key.pem":  endpoint.KeyFile,
	} {
		want, err := os.ReadFile(filepath.Join(certs, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		got, err := os.ReadFile(stored)
		if err != nil {
			t.Fatalf("reading stored %s: %v", name, err)
		}
		if string(got) != string(want) {
			t.Errorf("%s changed across the update", name)
		}
	}
}

// A description-only update must not touch the endpoint at all.
func TestUpdateContextEndpoint_DescriptionKeepsEndpoint(t *testing.T) {
	requireDocker(t)
	certs := writeTLSMaterial(t)
	createProbeContext(t, "tcp://127.0.0.1:2376", certs)

	if err := docker.UpdateContextEndpoint(probeContext, "renamed", ""); err != nil {
		t.Fatalf("UpdateContextEndpoint failed: %v", err)
	}

	endpoint, err := docker.InspectContextEndpoint(probeContext)
	if err != nil {
		t.Fatalf("InspectContextEndpoint failed: %v", err)
	}
	if endpoint.Host != "tcp://127.0.0.1:2376" {
		t.Errorf("host = %q, want it unchanged", endpoint.Host)
	}
	if !endpoint.HasTLS() {
		t.Errorf("the TLS material did not survive a description change: %+v", endpoint)
	}
}

// A context with no TLS material updates to a bare host.
func TestUpdateContextEndpoint_WithoutTLSMaterial(t *testing.T) {
	requireDocker(t)
	createProbeContext(t, "tcp://127.0.0.1:2375", "")

	if err := docker.UpdateContextEndpoint(probeContext, "", "ssh://user@10.0.0.7"); err != nil {
		t.Fatalf("UpdateContextEndpoint failed: %v", err)
	}

	endpoint, err := docker.InspectContextEndpoint(probeContext)
	if err != nil {
		t.Fatalf("InspectContextEndpoint failed: %v", err)
	}
	if endpoint.Host != "ssh://user@10.0.0.7" {
		t.Errorf("host = %q, want the updated endpoint", endpoint.Host)
	}
	if endpoint.HasTLS() {
		t.Errorf("invented TLS material: %+v", endpoint)
	}
}

func TestUpdateContextEndpoint_RefusesDefault(t *testing.T) {
	requireDocker(t)
	if err := docker.UpdateContextEndpoint(docker.DefaultContextName, "", "tcp://10.0.0.7:2376"); err == nil {
		t.Fatal("expected the built-in context to be refused")
	}
}

func requireDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available; skipping integration test")
	}
}

// createProbeContext creates the throwaway context, with TLS material when
// certsDir is set, and removes it when the test ends.
func createProbeContext(t *testing.T, host, certsDir string) {
	t.Helper()
	remove := func() { _ = exec.Command("docker", "context", "rm", "-f", probeContext).Run() }
	remove()

	endpoint := "host=" + host
	if certsDir != "" {
		endpoint += ",ca=" + filepath.Join(certsDir, "ca.pem")
		endpoint += ",cert=" + filepath.Join(certsDir, "cert.pem")
		endpoint += ",key=" + filepath.Join(certsDir, "key.pem")
	}
	out, err := exec.Command("docker", "context", "create", probeContext, "--docker", endpoint).CombinedOutput()
	if err != nil {
		t.Fatalf("creating the probe context failed: %v: %s", err, out)
	}
	t.Cleanup(remove)
}

// writeTLSMaterial writes a self-signed CA and client keypair, which is all
// Docker checks when it stores endpoint TLS material.
func writeTLSMaterial(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeSelfSigned(t, filepath.Join(dir, "ca.pem"), "", "probe-ca")
	writeSelfSigned(t, filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"), "probe-client")
	return dir
}

func writeSelfSigned(t *testing.T, certPath, keyPath, commonName string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating certificate: %v", err)
	}
	writePEM(t, certPath, &pem.Block{Type: "CERTIFICATE", Bytes: der})

	if keyPath == "" {
		return
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshalling key: %v", err)
	}
	writePEM(t, keyPath, &pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
}

func writePEM(t *testing.T, path string, block *pem.Block) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
	defer file.Close()
	if err := pem.Encode(file, block); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
