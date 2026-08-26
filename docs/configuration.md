<!--
SPDX-License-Identifier: Apache-2.0
Copyright © 2026 Eldara Tech
-->

# Configuration

This page is the single source of truth for SwarmCLI's environment variables and
on-disk paths. Rows marked **BE** apply to the Business Edition only; the rest
apply to every build.

## Environment variables

| Variable | Edition | Purpose | Default | Read at |
|---|---|---|---|---|
| `SWARMCLI_ENV` | both | `dev` writes human-readable logs, `prod` writes JSON. | `prod` | startup |
| `LOG_LEVEL` | both | Log verbosity: `debug`, `info`, `warn`, `error`. | `debug` in dev, `info` in prod | startup |
| `DOCKER_CONTEXT` | both | Docker context to talk to. It overrides `docker context use`, so while it is set the context switcher refuses to move to a different context rather than writing a switch that could not take effect. | the active context | startup |
| `SWARMCLI_DISABLE_VERSION_CHECK` | both | Disables the startup request to `https://swarmcli.io/api/v1/version` that checks whether a newer release is available. | unset | startup |
| `SWARMCLI_CHARTS_ALLOW_PLAINTEXT` | both | Allows chart repositories served over plain `http://`, which are refused by default (see [charts/README.md](../charts/README.md#transport)). | unset (https only) | `charts` commands |
| `SWARMCLI_CHARTS_NO_AUTO_UPDATE` | both | Stops a `charts` command refreshing a repository index before resolving a chart from it. `--no-repo-update` does the same for one invocation. | unset (refreshes) | `charts` commands |
| `EDITOR` | both | Editor invoked by the in-TUI edit actions (stack, config, secret). | `nano` | edit action |
| `XDG_STATE_HOME` | both | Base directory for logs and chart state — see [On-disk paths](#on-disk-paths). | `~/.local/state` | startup, `charts` commands |
| `SWARMCLI_LICENSE` | BE | License key. Takes priority over the license file. | unset | startup |
| `SWARMCLI_DISABLE_LICENSE_RENEWAL` | BE | Stops swarmcli asking the license service to renew a [managed license](license.md#managed-licenses-activation-is-a-second-step)'s activation. Leases are then installed from a file. No effect on any other license type, which never make the request at all. | unset | startup |
| `SWARMCLI_LICENSE_API_URL` | BE | Base URL of the license service renewals are asked of. Pointing it elsewhere cannot grant anything — every artifact is verified against a key compiled into swarmcli. | `https://swarmcli.io/api/v1` | startup |
| `SWARMCLI_PROXY_URL` | BE | WebSocket URL of the rbac-proxy. Auto-derived from the active Docker context if unset. | unset | shell connect |
| `SWARMCLI_REVEAL_IMAGE` | BE | Image used for the temporary service that reveals a secret. | `alpine:latest` | reveal action |
| `SWARMCLI_SHELL_CMD` | BE | Shell command to exec when opening a shell into a task. If unset, the agent auto-detects (`bash` → `sh` → `ash`). | unset | shell connect |
| `SWARMCLI_FORWARD_IDLE_TIMEOUT` | BE | Idle timeout for an active port-forward (no traffic in either direction). Accepts any Go duration; capped at `24h`. | `30m` | per-forward, evaluated continuously |

The four on/off variables (`SWARMCLI_DISABLE_VERSION_CHECK`,
`SWARMCLI_CHARTS_ALLOW_PLAINTEXT`, `SWARMCLI_CHARTS_NO_AUTO_UPDATE`,
`SWARMCLI_DISABLE_LICENSE_RENEWAL`) accept the
values Go's `strconv.ParseBool` does — `1`, `t`, `true`, `TRUE` and their false
counterparts. Anything else is treated as unset.

See [License — Activation](license.md#activation) for the precedence
between `SWARMCLI_LICENSE` and the license file, and
[Features](features.md) for how `SWARMCLI_REVEAL_IMAGE`,
`SWARMCLI_SHELL_CMD`, and `SWARMCLI_FORWARD_IDLE_TIMEOUT` are used.

## The Docker context

swarmcli resolves the Docker context once, at startup — from `DOCKER_CONTEXT`
if it is set, otherwise from the active context — and then addresses that one
context for the rest of the session. Everything it does goes to the same
swarm: the lists, the logs, `stack deploy`, and every `docker` command it runs
for you.

So running `docker context use <other>` in another terminal does **not** move a
running swarmcli. Instead, within a few seconds it asks:

```
The Docker context changed outside swarmcli: 'a' → 'b'.

swarmcli is still using 'a'. Switch to 'b'?
```

Answer `y` to follow the switch — swarmcli drops its connection, reconnects to
`b` and reloads the cluster. Answer `n` to stay on `a`; you are not asked about
`b` again unless the context changes to something else, or changes back to `a`
and away again. Switching from inside swarmcli, with `:contexts`, needs no
prompt and still runs `docker context use`, so your shell follows along.

## On-disk paths

| Path | Edition | Mode | Contents |
|---|---|---|---|
| `~/.local/state/swarmcli/app.log` | both | `0600` | JSON logs (`SWARMCLI_ENV=prod`). Rotated at 20 MB, 5 compressed backups, 14 days. |
| `~/.local/state/swarmcli/app-debug.log` | both | `0600` | Human-readable logs (`SWARMCLI_ENV=dev`), rotated the same way. |
| `~/.local/state/swarmcli/charts/repos.json` | both | `0644` | Chart repositories configured with `swarmcli charts repo add`. |
| `~/.local/state/swarmcli/charts/cache/index-<repo>.yaml` | both | `0644` | Cached repository index per configured repository. |
| `~/.local/state/swarmcli/charts/cache/charts/<sha256>.tgz` | both | `0644` | Chart archives already downloaded, kept under the sha256 their repository index publishes. Re-verified on every read, and swept 30 days after the last one. |
| `~/.config/swarmcli/update-notice.json` | both | `0644` | The release version at which the startup update notice was dismissed. |
| `~/.config/swarmcli/license.key` | BE | `0600` | Active license key. Created by the startup prompt or by `:license <s>`. |
| `~/.config/swarmcli/certs/<stack>/ca.pem` | BE | `0600` | CA cert from `:bootstrap`. |
| `~/.config/swarmcli/certs/<stack>/cert.pem` | BE | `0600` | Admin client cert (CN = seed username). |
| `~/.config/swarmcli/certs/<stack>/key.pem` | BE | `0600` | Admin client private key. |
| `~/.docker/contexts/…` | BE | (Docker default) | Managed Docker contexts — the `<original>-managed` entry created by `:bootstrap` and any `<user>-managed` entries imported via `docker context import`. |

Everything under `~/.local/state/swarmcli/` moves with `XDG_STATE_HOME` when
that is set, and falls back to the system temp directory when there is no home
directory. Directory mode is `0755` there, and `0700` for
`~/.config/swarmcli/certs/<stack>/`. `<stack>` is the bootstrap stack name;
default `swarmcli-infra`.

Release state is not on disk: `swarmcli charts` stores each release's values and
rendered manifest in the swarm itself, as Docker configs.

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
