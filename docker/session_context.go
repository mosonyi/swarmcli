// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// envContextVar is the environment variable Docker resolves ahead of the
	// config file (docker/cli cli/command.EnvOverrideContext).
	envContextVar = "DOCKER_CONTEXT"

	// envHostVar forces Docker onto the `default` context, ahead of both the
	// config file and DOCKER_CONTEXT.
	envHostVar = "DOCKER_HOST"

	// envConfigDirVar names the directory holding the CLI's config.json
	// (docker/cli cli/config.EnvOverrideConfigDir).
	envConfigDirVar = "DOCKER_CONFIG"

	// defaultContextName is what Docker resolves to when nothing names a
	// context.
	defaultContextName = "default"
)

// envContext reports the context named by the environment, if any, trimmed.
// The pin, ValidateContext's refusal guard and the drift check all read it
// through here, so they cannot disagree about what "pinned by the environment"
// means — including for a value that is only whitespace, which names nothing.
func envContext() string { return strings.TrimSpace(os.Getenv(envContextVar)) }

// The Docker context this process addresses, resolved once.
//
// swarmcli used to answer "which context?" twice, differently: the SDK client
// resolved it on first use and cached the connection for the life of the
// process, while every other caller shelled out to `docker context show` again
// and re-read ~/.docker/config.json. A `docker context use` in another terminal
// moved the second answer and not the first, so the lists kept describing one
// swarm while `stack deploy`, `stack rm` and the raw log fallbacks addressed
// another — see #611.
//
// The pin is the single answer. It does not follow the config file: a context
// switch made outside swarmcli must not move a running session between swarms
// under the operator's feet. The app watches for that drift and offers the
// switch instead (app/contextdrift.go).
var (
	sessionCtxMu sync.RWMutex
	sessionCtx   string
)

// activeContextFn is the seam for "what context is active right now", so the
// pin's behaviour is testable without a Docker config file on disk.
var activeContextFn = activeContextName

// InitSessionContext resolves the session context and pins it, returning the
// pinned name. Calling it again returns the existing pin without re-resolving,
// so the entry points may both call it (the TUI does, and so does the
// non-interactive CLI) without one of them winning a race.
func InitSessionContext() (string, error) {
	return SessionContext()
}

// SessionContext returns the pinned Docker context, resolving it on first call.
//
// Resolution order is Docker's own, minus the parts a library cannot see:
// $DOCKER_CONTEXT, then the active context from the config file. Note that
// Docker resolves $DOCKER_HOST ahead of $DOCKER_CONTEXT and lets it force the
// `default` context; swarmcli has never honoured $DOCKER_HOST — it always
// builds its client from a context's stored endpoint — which is why every
// shell-out names the pin with an explicit `--context`, the one thing that
// outranks $DOCKER_HOST.
func SessionContext() (string, error) {
	sessionCtxMu.RLock()
	pinned := sessionCtx
	sessionCtxMu.RUnlock()
	if pinned != "" {
		return pinned, nil
	}

	sessionCtxMu.Lock()
	defer sessionCtxMu.Unlock()
	// Another goroutine may have resolved it while this one waited.
	if sessionCtx != "" {
		return sessionCtx, nil
	}

	name, err := resolveContextName()
	if err != nil {
		return "", err
	}
	sessionCtx = name
	return sessionCtx, nil
}

// SetSessionContext moves the pin. It is how a switch made inside swarmcli —
// from the contexts view, or by accepting the drift prompt — takes effect: the
// client cache must be dropped separately with ResetClient, because a pinned
// name and a live connection are two different pieces of state.
func SetSessionContext(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	sessionCtxMu.Lock()
	sessionCtx = name
	sessionCtxMu.Unlock()
}

// EnvPinsContext reports whether DOCKER_CONTEXT names the context.
//
// While it does, ~/.docker/config.json cannot move this session — Docker
// resolves the variable ahead of the file — so the two naming different
// contexts is the documented arrangement rather than a switch to react to.
// ValidateContext refuses to move off it for the same reason.
func EnvPinsContext() bool {
	return envContext() != ""
}

// ResetSessionContext drops the pin, so the next SessionContext call resolves
// afresh.
//
// For tests. The pin is deliberately sticky — that is the whole point of it —
// which is exactly what makes a test that changes DOCKER_CONTEXT or the active
// context read a stale answer without this. Production moves the pin with
// SetSessionContext, which names the new context rather than re-reading the
// config file.
func ResetSessionContext() {
	sessionCtxMu.Lock()
	sessionCtx = ""
	sessionCtxMu.Unlock()
}

// ConfigFileContext reports the context Docker itself would use right now,
// reading past the pin. It is what the drift check compares against, and the
// only caller that wants the live answer rather than the session's.
func ConfigFileContext() (string, error) {
	return resolveContextName()
}

// resolveContextName performs the lookup the pin caches.
func resolveContextName() (string, error) {
	if ctxName := envContext(); ctxName != "" {
		return ctxName, nil
	}
	return activeContextFn()
}

// activeContextName reports the context `docker context show` would print,
// computed here rather than by running it.
//
// It used to fork the docker CLI. That was affordable while it happened once,
// at startup, to resolve the pin — but the drift check calls it on the app tick
// too, so since the tick learned to re-arm it is a process start every five
// seconds for the life of the session, to read one field. Everything that field
// depends on is readable directly: $DOCKER_HOST forces `default`, and otherwise
// the answer is `currentContext` from the CLI's config file.
//
// The failure modes follow Docker's, so swarmcli is never more broken than the
// CLI is for the same file. A missing config file is a machine where nobody has
// run `docker context use`, and Docker answers `default` there too. A malformed
// one Docker warns about and then answers `default` anyway — verified against
// `docker context show`, which exits 0 — so this warns and does the same rather
// than failing a session the CLI would not. Only a file that exists and cannot
// be read is an error, which is where Docker stops as well.
func activeContextName() (string, error) {
	// Docker resolves $DOCKER_HOST ahead of everything else and pins the
	// `default` context to it. swarmcli does not honour the variable — it
	// always builds its client from a context's stored endpoint, see
	// SessionContext — but it must still agree with Docker about the *name*,
	// or the drift check would report a switch that nobody made.
	if strings.TrimSpace(os.Getenv(envHostVar)) != "" {
		return defaultContextName, nil
	}

	path, err := dockerConfigPath()
	if err != nil {
		return defaultContextName, nil // nowhere to look is the same as nothing to find
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return defaultContextName, nil
	}
	if err != nil {
		return "", fmt.Errorf("reading Docker config %s: %w", path, err)
	}

	var cfg struct {
		CurrentContext string `json:"currentContext"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		l().Infof("Docker config %s could not be parsed (%v); using the %q context, as docker does", path, err, defaultContextName)
		return defaultContextName, nil
	}
	if name := strings.TrimSpace(cfg.CurrentContext); name != "" {
		return name, nil
	}
	return defaultContextName, nil
}

// dockerConfigPath locates the CLI's config file the way Docker does:
// $DOCKER_CONFIG names the directory, otherwise ~/.docker.
func dockerConfigPath() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(envConfigDirVar)); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docker", "config.json"), nil
}
