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
