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
the full release lifecycle — install, upgrade, rollback, uninstall, list, status,
history, diff, get, and prune — and declarative releases (`apply`, `outdated`).
Chart-dev tooling (`create`, `lint`, `dependency`) and subchart resolution are the
remaining phase.

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

A chart reference is either a configured `repo/chart` or a local path to a chart
directory or packaged `.tgz`. **`--version` selects a version from a repository
index, so it applies only to a `repo/chart` reference** — a local chart carries its
version in its own `Chart.yaml`, and passing `--version` with one is an error
rather than being silently ignored.

## Declarative releases (GitOps)

The commands above are imperative, and release state lives in the swarm — so
nothing in git says what *should* be running. `charts apply` closes that gap: it
converges the swarm to a file you commit.

```yaml
# swarmcli-release.yaml
apiVersion: v1
owner: prod-swarm              # optional; see Ownership below
repositories:
  - name: swarmcli-charts
    url: https://eldara-tech.github.io/swarmcli-charts
releases:
  - name: edge
    chart: swarmcli-charts/traefik
    version: "0.1.1"
    values: [./traefik.yaml]     # relative to THIS FILE, not the working directory
  - name: hello
    chart: swarmcli-charts/whoami
    version: "0.1.8"
```

```bash
swarmcli charts apply -f swarmcli-release.yaml --dry-run   # plan
swarmcli charts apply -f swarmcli-release.yaml --diff      # plan + manifest diffs
swarmcli charts apply -f swarmcli-release.yaml             # converge
swarmcli charts outdated                                   # what has a newer chart?
```

| Behaviour | |
|---|---|
| Missing release | installed |
| Changed chart version, values, or rendered manifest | upgraded |
| Identical | **skipped — no new revision** |
| On the swarm but not in the file | **reported, never removed** |
| Installed by this file, no longer in it | **reported as an orphan, still never removed** |

Two of those deserve the emphasis. Releases are **never deleted** — apply prints
the `uninstall` command and leaves the decision to you. And an unchanged release
is skipped **entirely**: history is one Docker Config per revision, so re-applying
on every CI push would otherwise grow the swarm's config store without bound.

### Ownership

`owner:` names the manifest, and every release it installs is stamped with that
name — in the release-history Config's `com.swarmcli.owner` label and in the
stored record — so a later apply can tell a release *this file* installed from
one it has simply never seen. Dropping a release from the file then reports it as
an **orphan**: provably obsolete, because the stamp says this manifest produced
it and nothing else claims it. A release with no stamp, or another manifest's
stamp, stays merely *unmanaged*.

Nothing is deleted either way. The distinction is the prerequisite for a prune
that could be: it is what separates "this is obsolete" from "I do not recognise
this", and only the first is ever safe to act on.

```
com.swarmcli.owner: apply/prod-swarm:release/hello
                    └──── id ─────┘ └── resource ──┘
```

The stamp names the **resource** as well as the owner, and both halves must match
for it to count. A bare owner string cannot tell a release this file installed
from a copy of one — ArgoCD shipped exactly that as `app.kubernetes.io/instance`
and replaced it in 3.0 for the same reason. The `apply/` prefix keeps a manifest
applied from the command line from colliding with a controller that happened to
pick the same name.

`owner:` is optional and has **no default**. A derived one would either change
between a laptop and a CI checkout (a path hash) or be shared by every repository
using the conventional filename (a basename) — and either would let two unrelated
manifests claim each other's releases. Omit it and nothing is claimed, which is
exactly the behaviour of every version before this one.

For a `repo/chart`, **`version` is required** — a floating pin would silently
upgrade production on the next `apply`. For a **local** chart path (`chart:
./charts/mine`, resolved against the file) it must be **omitted**: the chart's own
`Chart.yaml` sets the version, so there is nothing to select.

Unknown keys are rejected, so a typo fails loudly instead of quietly doing nothing.
Releases are applied in file order; add `--wait` if a later release needs an
earlier one live.

`apply` honours `--wait`, `--timeout` and `--history-max`. It **rejects** `--set`,
`--version`, `--reuse-values`, `--install`, `--purge-volumes`, `--requirements` and
`--revision` rather than ignoring them — the file is the only source of truth, so a
value passed on the command line would be a lie. `--diff` implies `--dry-run` and
never deploys.

### Keeping it up to date automatically

If your charts come from [swarmcli-charts], extend its Renovate preset — that is
the whole configuration:

```json
{ "extends": ["github>Eldara-Tech/swarmcli-charts"] }
```

For any other chart repository, one line does the same job:

```json
{ "helmfile": { "managerFilePatterns": ["/(^|/)swarmcli-release\\.ya?ml$/"] } }
```

Both work because the file's key names match [Helmfile](https://helmfile.readthedocs.io/)'s,
so [Renovate](https://docs.renovatebot.com/)'s **built-in** `helmfile` manager reads
it — no custom regex to maintain. Renovate resolves each chart against the
`repositories` you declared and opens a PR bumping `version:` when a new chart
version is published, with the chart's release notes attached. Merge it, and
`swarmcli charts apply` in CI rolls it out.

[swarmcli-charts]: https://github.com/Eldara-Tech/swarmcli-charts

## Chart format

```
mychart/
├── Chart.yaml          # apiVersion, name, version, appVersion, swarmcliVersion, …
├── values.yaml         # default values
├── values.schema.json  # optional JSON Schema validated before render
├── README.md
└── templates/          # Go-templated Compose fragments
    ├── stack.yaml
    ├── configs.yaml
    ├── secrets.yaml
    └── volumes.yaml
```

`apiVersion` is `v1` (the only format this build reads; absent means `v1`).

### Declaring the swarmcli a chart needs

A chart may state the chart engine it requires, as a SemVer constraint:

```yaml
# Chart.yaml
swarmcliVersion: ">= 1.13.0"
```

`install`, `upgrade` and `apply` refuse a chart this build is too old for, so the
failure names the version to upgrade to rather than surfacing as whatever error
the missing feature happens to produce (`function "toYamlPretty" not defined`).
`install` and `upgrade` ask before refusing when run interactively; `apply` never
does — it is meant to run unattended. `template`, `diff` and `show` only warn:
they change nothing, and `show` is how you find out what a chart wants. Pass
`--skip-compat-check` to proceed anyway.

The constraint is checked against the **chart engine's** version, which is not
necessarily the version the binary reports for itself — a binary embedding this
package carries whichever engine it pinned. A build that reports no engine
version (any `go build`) warns rather than refusing: not knowing is not evidence
of incompatibility. Charts declaring nothing are unaffected.

Note that only builds carrying this check can honour it: `Chart.yaml` is parsed
leniently, so an older swarmcli silently ignores `swarmcliVersion` — the same
bootstrapping limit Helm's own `apiVersion` gate has.

### Linting a chart

```bash
swarmcli charts lint ./mychart                       # against this build
swarmcli charts lint ./mychart -f ./ci/default-values.yaml
swarmcli charts lint ./mychart --for-version 1.12.0  # against another version
```

`lint` renders the chart and reports every problem it finds — a broken template,
values that fail `values.schema.json`, a `swarmcliVersion` this build does not
satisfy — rather than stopping at the first. It renders from the chart defaults,
layering any `-f`/`--set` on top: a chart with a required, undefaulted input (a
`{{ required }}` / `{{ fail }}` guard) cannot render from bare defaults, so lint
it with the values a real install would supply. A chart that declares no
`swarmcliVersion` gets a warning, not an error: the field is optional, but a
chart naming no floor leaves an operator on an old build nothing to act on.

`--for-version` asks **whether the chart's declared floor admits that version**.
It cannot tell you the chart *runs* on it: this binary carries one engine's
behaviour and cannot emulate another's, so it checks the claim's shape, not its
truth. Rendering with a real binary of that version is the only thing that
settles it — which is what a chart repository's CI should do:

```bash
SWARMCLI_REF=v1.10.0 scripts/install-swarmcli.sh ./bin  # build the declared floor
./bin/swarmcli charts template t ./mychart              # prove the chart runs on it
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

### Integrity

The index is fetched over HTTPS from the repository, but a chart's tarball URL may
point anywhere — a GitHub Release asset, a CDN. The `digest` the index publishes
for each version is what binds the two together, so swarmcli **verifies it**:

| Index entry | Behaviour on `install`/`upgrade`/`template` |
|---|---|
| `digest: sha256:<hex>` matches the download | installs |
| digest does **not** match | **fatal** — `digest mismatch … refusing to install` |
| digest uses an algorithm other than sha256 | **fatal** — verification cannot be performed, so it is not skipped |
| entry publishes **no** digest | installs, with a warning on stderr |

An absent digest warns rather than fails because nothing was verified before this
existed, so rejecting would break every repository that publishes none — including
older and hand-written ones. Both the bare-hex form (`helm repo index`) and the
`sha256:`-prefixed form are accepted.

Chart archives are also capped at 20 MiB on the wire (the decompressed contents
have their own limits), so a hostile or truncated download cannot exhaust memory
before it is hashed.

TLS is not sufficient on its own here: it authenticates the *host* you downloaded
from, not that the bytes are the ones the repository index vouched for.

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
- **Chart integrity** is only as good as the index: a repository that publishes
  no `digest` gets a warning, not a refusal (see [Integrity](#integrity)). Chart
  archives are capped at 20 MiB on the wire.
- `docker stack deploy` is used under the hood, so only the Compose-on-Swarm
  subset is supported and the `docker` CLI must be on `PATH`.

[issue #413]: https://github.com/Eldara-Tech/swarmcli/issues/413
[Sprig]: https://masterminds.github.io/sprig/
