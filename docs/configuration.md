<!--
SPDX-License-Identifier: Apache-2.0
Copyright © 2026 Eldara Tech
-->

# Configuration

This page is the single source of truth for Business Edition environment
variables and on-disk paths. For inherited Community Edition variables
(`SWARMCLI_ENV`, `LOG_LEVEL`, `DOCKER_CONTEXT`, `SWARMCLI_DISABLE_VERSION_CHECK`),
see the
[README](../README.md#environment-variables).

## BE-specific environment variables

| Variable | Purpose | Default | Read at |
|---|---|---|---|
| `SWARMCLI_LICENSE` | License key. Takes priority over the license file. | unset | startup |
| `SWARMCLI_PROXY_URL` | WebSocket URL of the rbac-proxy. Auto-derived from the active Docker context if unset. | unset | shell connect |
| `SWARMCLI_REVEAL_IMAGE` | Image used for the temporary service that reveals a secret. | `alpine:latest` | reveal action |
| `SWARMCLI_SHELL_CMD` | Shell command to exec when opening a shell into a task. If unset, the agent auto-detects (`bash` → `sh` → `ash`). | unset | shell connect |
| `SWARMCLI_FORWARD_IDLE_TIMEOUT` | Idle timeout for an active port-forward (no traffic in either direction). Accepts any Go duration; capped at `24h`. | `30m` | per-forward, evaluated continuously |

See [License — Activation](license.md#activation) for the precedence
between `SWARMCLI_LICENSE` and the license file, and
[Features](features.md) for how `SWARMCLI_REVEAL_IMAGE`,
`SWARMCLI_SHELL_CMD`, and `SWARMCLI_FORWARD_IDLE_TIMEOUT` are used.

## On-disk paths

| Path | Mode | Contents |
|---|---|---|
| `~/.config/swarmcli/license.key` | `0600` | Active license key. Created by the startup prompt or by `:license <s>`. |
| `~/.config/swarmcli/certs/<stack>/ca.pem` | `0600` | CA cert from `:bootstrap`. |
| `~/.config/swarmcli/certs/<stack>/cert.pem` | `0600` | Admin client cert (CN = seed username). |
| `~/.config/swarmcli/certs/<stack>/key.pem` | `0600` | Admin client private key. |
| `~/.docker/contexts/…` | (Docker default) | Managed Docker contexts — the `<original>-managed` entry created by `:bootstrap` and any `<user>-managed` entries imported via `docker context import`. |

Directory mode is `0700` for `~/.config/swarmcli/certs/<stack>/`.
`<stack>` is the bootstrap stack name; default `swarmcli-infra`.

Log paths are unchanged from CE — see the
[README](../README.md).

## Stack-side configuration (rbac-proxy)

The `:bootstrap` command renders an embedded stack template. A handful of
proxy-side environment variables are set on the rbac-proxy service and
are useful to know when operating the deployment:

| Variable | Purpose | Set by bootstrap |
|---|---|---|
| `PROXY_LISTEN` | mTLS listen address. | `:<port>` (default `:2376`). |
| `PROXY_INTERNAL_LISTEN` | Plaintext loopback listener for management calls. | `127.0.0.1:2375`. |
| `PROXY_DOCKER_SOCKET` | Backend socket. | `/var/run/docker.sock`. |
| `PROXY_AGENT_MANAGER_URL` | Where the proxy reaches the agent-manager. | `tcp://swarmcli-infra_agent-manager:8080`. |
| `PROXY_STORE` / `PROXY_DATABASE_PATH` | User/role store. | `sqlite` at `/data/proxy.db` (volume `proxy-data`). |
| `PROXY_TLS_*` | TLS material — server cert/key, client CA cert/key. | Mounted from Docker secrets. |
| `PROXY_ADMIN_TOKEN_FILE` | Bearer token for the management API. | Mounted from the `_admin-token` secret. |
| `PROXY_EXTERNAL_URL` | URL embedded into onboarding bundles. | `tcp://<host>:<port>`. |
| `PROXY_SEED_USERNAME` / `PROXY_SEED_ROLE` | Bootstrap admin user. | `admin` / `admin`. |
| `PROXY_ONBOARDING_TOKEN_TTL` | Time-to-live for onboarding tokens. | Default `24h`. |

Changing these requires editing the deployed service (`docker service update`)
or re-running `:bootstrap` from scratch — see
[Bootstrap — Re-bootstrap](bootstrap.md#re-bootstrap-and-teardown).

## Stack-side configuration (agent)

One agent-side variable is worth knowing when using [Volumes](volumes.md):

| Variable | Purpose | Set by bootstrap |
|---|---|---|
| `FILEOP_MAX_UPLOAD_BYTES` | Maximum size of a single volume file/directory upload; the agent rejects a larger payload with `413`. | `2147483648` (2 GiB). |

Deployments bootstrapped before this was templated fall back to the agent's
built-in 2 GiB default. Raise or lower it the same way as the proxy-side
variables above — `docker service update` the agent service, or re-run
`:bootstrap`.
