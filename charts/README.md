<!--
SPDX-License-Identifier: Apache-2.0
Copyright © 2026 Eldara Tech
-->

# SwarmCLI Charts

A Helm-inspired package manager for Docker Swarm. Charts package Docker Stack
(Compose) templates with default values and metadata; installing a chart
produces a **release** — a Docker stack whose revision history is stored in
Docker Configs.

Implemented so far ([issue #413]): repository management, discovery, templating,
and the full release lifecycle — install, upgrade, rollback, uninstall, list,
status, history, diff, and get. Chart-dev tooling (`create`, `lint`,
`dependency`) and subchart resolution are the remaining phase.

## Invocation

Charts run through swarmcli's **non-interactive CLI**: when the binary is given
arguments it executes a one-shot command and exits (a bare `swarmcli` still
launches the TUI). This makes charts scriptable for CI/CD and GitOps.

```bash
swarmcli charts repo add eldara https://charts.example.com
swarmcli charts repo update
swarmcli charts search traefik
swarmcli charts show values eldara/traefik > values.yaml
swarmcli charts template my-traefik eldara/traefik -f values.yaml
swarmcli charts install  my-traefik eldara/traefik -f values.yaml
swarmcli charts status   my-traefik
swarmcli charts diff upgrade my-traefik eldara/traefik --set replicas=3
swarmcli charts upgrade  my-traefik eldara/traefik --set replicas=3
swarmcli charts history  my-traefik
swarmcli charts rollback my-traefik 1
swarmcli charts list
swarmcli charts uninstall my-traefik
```

A chart reference is either a configured `repo/chart` (optionally `--version`)
or a local path to a chart directory or packaged `.tgz`.

## Chart format

```
mychart/
├── Chart.yaml          # name, version, appVersion, description, maintainers
├── values.yaml         # default values
├── values.schema.json  # optional JSON Schema validated before render
├── README.md
└── templates/          # Go-templated Compose fragments
    ├── stack.yaml
    ├── configs.yaml
    ├── secrets.yaml
    └── volumes.yaml
```

Templates are rendered with Go `text/template` + [Sprig], exposing:

- `.Values` — merged values (defaults < `-f` files < `--set`)
- `.Release` — `.Name`, `.Namespace`, `.Revision`
- `.Chart` — `.Name`, `.Version`, `.AppVersion`

Each `templates/*.yaml` is rendered then **deep-merged** into a single Compose
document (Compose is one document, unlike Helm's concatenated manifests). Files
beginning with `_` (e.g. `_helpers.tpl`) define named templates only. The merged
manifest is validated as a Docker stack before use.

## Repositories

A repository is an HTTP(S)-served `index.yaml` listing chart versions, each with
a tarball URL (Helm repository format) — hostable on GitHub Pages/Releases, S3,
or any static host. Configured repos and cached indexes live under
`$XDG_STATE_HOME/swarmcli/charts` (default `~/.local/state/swarmcli/charts`).

## Release storage

Each release revision is stored as an immutable, gzipped Docker Config named
`swarmcli.release.<release>.v<N>`, labeled `com.swarmcli.*`. The log is
append-only: the highest revision is the current state; lower deployed revisions
display as `superseded`. This gives HA (Swarm Raft) and rollback with no
external database.

```bash
docker config ls --filter label=com.swarmcli.release=my-traefik
```

## Pruning release history

Because every revision is its own Config, history grows without bound. Trim it
with the retention window — keep only the newest `N` revisions:

```bash
# Apply a window inline, after the deploy:
swarmcli charts install my-traefik eldara/traefik --history-max 20
swarmcli charts upgrade my-traefik eldara/traefik --history-max 20

# Or prune existing history on demand:
swarmcli charts prune my-traefik --history-max 20   # one release
swarmcli charts prune --history-max 20              # every release
swarmcli charts prune my-traefik --history-max 20 --dry-run  # preview only
```

`prune` deletes the oldest revisions beyond the window and **always keeps the
current (highest) revision**, so the live release and rollback targets inside the
window are preserved. Without `--history-max` (or with `0`) it keeps everything
and reports that no window was given. `--dry-run` prints the keep/delete decision
without touching Docker.

> **Use `swarmcli charts prune`, not raw Docker.** `docker config prune`,
> `docker system prune` and `docker config rm swarmcli.release.*` are not
> SwarmCLI-aware and will corrupt release history, rollback targets and audit
> lineage. SwarmCLI is the only sanctioned way to delete a release's resources.

Per-revision protection labels (`com.swarmcli.keep`, `com.swarmcli.protected`)
are a planned Phase-3 addition; today the current revision plus the newest-`N`
window are the protections.

## Notes & limitations

- **`install --dry-run`** renders, validates, and computes the next revision
  but does not deploy. For fully offline rendering use `charts template`.
- **`--purge-volumes`** removes volumes on the connected node only (the CE
  single-node volume scope); cross-node purge is a future extension.
- **Secrets:** the rendered manifest is stored in a Docker Config, which is
  readable by anyone with Docker access. Do **not** inline secret material in
  templates — reference Docker secrets as separate objects instead. A redaction
  pass is planned before charts ship broadly.
- `docker stack deploy` is used under the hood, so only the Compose-on-Swarm
  subset is supported and the `docker` CLI must be on `PATH`.

[issue #413]: https://github.com/Eldara-Tech/swarmcli/issues/413
[Sprig]: https://masterminds.github.io/sprig/
