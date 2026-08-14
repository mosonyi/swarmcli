// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"errors"
	"strings"
	"testing"
)

// Captured from `docker context inspect`, so the shape these tests read is the
// one Docker writes rather than one we remembered.
const (
	inspectTLSContext = `[{"Name":"probe","Metadata":{"Description":"probe"},` +
		`"Endpoints":{"docker":{"Host":"tcp://10.0.0.7:2376","SkipTLSVerify":true}},` +
		`"TLSMaterial":{"docker":["ca.pem","cert.pem","key.pem"]},` +
		`"Storage":{"MetadataPath":"/home/u/.docker/contexts/meta/93f5","TLSPath":"/home/u/.docker/contexts/tls/93f5"}}]`

	inspectPlainContext = `[{"Name":"plain","Metadata":{},` +
		`"Endpoints":{"docker":{"Host":"ssh://user@10.0.0.7","SkipTLSVerify":false}},` +
		`"TLSMaterial":{},"Storage":{"MetadataPath":"/home/u/.docker/contexts/meta/aa01","TLSPath":"/home/u/.docker/contexts/tls/aa01"}}]`

	inspectDefaultContext = `[{"Name":"default","Metadata":{},` +
		`"Endpoints":{"docker":{"Host":"unix:///var/run/docker.sock","SkipTLSVerify":false}},` +
		`"TLSMaterial":{},"Storage":{"MetadataPath":"<IN MEMORY>","TLSPath":"<IN MEMORY>"}}]`
)

func TestParseContextEndpoint_TLS(t *testing.T) {
	endpoint, err := parseContextEndpoint("probe", inspectTLSContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint.Host != "tcp://10.0.0.7:2376" {
		t.Errorf("host = %q", endpoint.Host)
	}
	if !endpoint.SkipTLSVerify {
		t.Error("SkipTLSVerify was dropped")
	}
	if !endpoint.HasTLS() {
		t.Fatalf("expected TLS material, got %+v", endpoint)
	}
	// The store's own copies are the only reference an update can re-supply.
	if endpoint.CAFile != "/home/u/.docker/contexts/tls/93f5/docker/ca.pem" {
		t.Errorf("ca = %q", endpoint.CAFile)
	}
	if endpoint.CertFile != "/home/u/.docker/contexts/tls/93f5/docker/cert.pem" {
		t.Errorf("cert = %q", endpoint.CertFile)
	}
	if endpoint.KeyFile != "/home/u/.docker/contexts/tls/93f5/docker/key.pem" {
		t.Errorf("key = %q", endpoint.KeyFile)
	}
}

func TestParseContextEndpoint_NoTLSMaterial(t *testing.T) {
	for name, inspectJSON := range map[string]string{
		"ssh":     inspectPlainContext,
		"default": inspectDefaultContext,
	} {
		t.Run(name, func(t *testing.T) {
			endpoint, err := parseContextEndpoint(name, inspectJSON)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if endpoint.HasTLS() {
				t.Errorf("invented TLS material: %+v", endpoint)
			}
			if endpoint.CAFile != "" || endpoint.CertFile != "" || endpoint.KeyFile != "" {
				t.Errorf("expected no cert paths, got %+v", endpoint)
			}
		})
	}
}

func TestParseContextEndpoint_Errors(t *testing.T) {
	if _, err := parseContextEndpoint("broken", "not json"); err == nil {
		t.Error("expected an error for unparseable output")
	}
	if _, err := parseContextEndpoint("missing", "[]"); err == nil {
		t.Error("expected an error for an empty inspect result")
	}
}

// A host change must restate the TLS material: `docker context update`
// replaces the whole endpoint and resets the stored material to exactly what
// the argv names, so a bare host= deletes a TLS context's certificates.
func TestUpdateContextArgs_HostCarriesTLSMaterial(t *testing.T) {
	endpoint := ContextEndpoint{
		Host:          "tcp://old:2376",
		SkipTLSVerify: true,
		CAFile:        "/store/ca.pem",
		CertFile:      "/store/cert.pem",
		KeyFile:       "/store/key.pem",
	}
	args := updateContextArgs("prod", "", "tcp://new:2376", endpoint)

	config := dockerFlagValue(t, args)
	for _, want := range []string{
		"host=tcp://new:2376",
		"ca=/store/ca.pem",
		"cert=/store/cert.pem",
		"key=/store/key.pem",
		"skip-tls-verify=true",
	} {
		if !strings.Contains(config, want) {
			t.Errorf("--docker %q is missing %q", config, want)
		}
	}
}

func TestUpdateContextArgs_HostWithoutTLSMaterial(t *testing.T) {
	args := updateContextArgs("prod", "", "ssh://user@new", ContextEndpoint{Host: "ssh://user@old"})
	if config := dockerFlagValue(t, args); config != "host=ssh://user@new" {
		t.Errorf("--docker = %q, want a bare host", config)
	}
}

// Without a host change there is no --docker, which is what leaves an existing
// endpoint — and its TLS material — untouched.
func TestUpdateContextArgs_DescriptionOnly(t *testing.T) {
	endpoint := ContextEndpoint{
		Host:     "tcp://old:2376",
		CAFile:   "/store/ca.pem",
		CertFile: "/store/cert.pem",
		KeyFile:  "/store/key.pem",
	}
	args := updateContextArgs("prod", "staging swarm", "", endpoint)

	for i, arg := range args {
		if arg == "--docker" {
			t.Fatalf("description-only update passed --docker %q", args[i+1])
		}
	}
	if got := flagValue(args, "--description"); got != "staging swarm" {
		t.Errorf("--description = %q", got)
	}
}

// Docker ignores an empty --description, so passing one would be a request
// that silently does nothing.
func TestUpdateContextArgs_EmptyDescriptionOmitted(t *testing.T) {
	args := updateContextArgs("prod", "", "", ContextEndpoint{})
	for _, arg := range args {
		if arg == "--description" {
			t.Fatal("an empty description must not be passed to Docker")
		}
	}
	if strings.Join(args, " ") != "context update prod" {
		t.Errorf("args = %v", args)
	}
}

func TestUpdateContextEndpoint_Rejects(t *testing.T) {
	if err := UpdateContextEndpoint("", "", "tcp://new:2376"); err == nil {
		t.Error("expected an error for an empty name")
	}
	err := UpdateContextEndpoint(DefaultContextName, "", "tcp://new:2376")
	if !errors.Is(err, ErrDefaultContextImmutable) {
		t.Errorf("err = %v, want ErrDefaultContextImmutable", err)
	}
}

// dockerFlagValue returns the single --docker value, failing if there is none.
func dockerFlagValue(t *testing.T, args []string) string {
	t.Helper()
	value := flagValue(args, "--docker")
	if value == "" {
		t.Fatalf("no --docker in %v", args)
	}
	return value
}

func flagValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
