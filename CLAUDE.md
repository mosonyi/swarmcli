# SwarmCLI

Keyboard-driven TUI for Docker Swarm management ("k9s for Docker Swarm"). Single static Go binary, Bubble Tea framework, zap logging.

## Quick Reference

```bash
# Build & run
go build -v -o swarmcli .
SWARMCLI_ENV=dev LOG_LEVEL=debug go run .

# Unit tests
go test ./...

# Integration tests (full E2E against real Docker Swarm via DinD)
./test-setup/testenv.sh integration           # up → deploy → test → down
./test-setup/testenv.sh up                    # start Swarm environment only
./test-setup/testenv.sh deploy                # deploy test stack
./test-setup/testenv.sh test                  # run integration tests
./test-setup/testenv.sh test TestScaleWhoami  # single test
./test-setup/testenv.sh down                  # teardown + cleanup
KEEP=1 ./test-setup/testenv.sh integration    # keep env running after tests
TEST_LOG=1 ./test-setup/testenv.sh test       # enable test logging

# Lint (CI uses this)
golangci-lint run ./... --build-tags=integration

# Logs
tail -f ~/.local/state/swarmcli/app-debug.log   # dev mode
tail -f ~/.local/state/swarmcli/app.log          # prod mode (JSON)
```

## Architecture

```
main.go                    Entry point; version injection via ldflags. With args, dispatches non-interactive CLI subcommands via cli.Dispatch(); bare invocation launches the TUI (tea.NewProgram())
cli/                       Arg-based CLI dispatch (cli.Dispatch): `charts`, `version`, `help`; cli/apply.go holds the GitOps subcommands (`charts apply`, `charts outdated`)
charts/                    Helm-like package manager (repos, chart rendering, releases) + declarative releases (releasefile.go, apply.go, outdated.go). charts.ChartSource (source.go) is the seam that resolves a chart ref — repo or local path — so release planning is testable without Docker, a network or a filesystem. charts.NewDockerBackend(ctxName) + NewEngineWith is the seam that targets a *specific* swarm: the default backend uses the ambient Docker context, the SDK client singleton and the shared snapshot cache, all three of which are process-global
app/
  app.go                   Init(); triggers command autoload via _ "github.com/Eldara-Tech/swarmcli/commands" and view autoload via _ "github.com/Eldara-Tech/swarmcli/views" (view factory registry lives in views/view/registry.go)
  hooks.go                 PreUpdateHook registration; StartupOverlay; RegisterShutdownHook / RunShutdownHooks (BE port-forward manager registers CloseAll here)
  model.go                 Central state: Model struct (viewport, currentView, viewStack, commandInput, searchInput, systemInfo)
  update.go                Main message router: navigation, resize, events, key dispatch
views/
  view/interface.go        View contract: Update/View/Init/Name/OnEnter/OnExit/HasErrors/ShortHelpItems + Filterable interface
  stacks/                  Stack list → drill into services
  services/                Service list (filterable by stack/node/all), scale/restart actions
  tasks/                   Task list per service/stack
  nodes/                   Cluster node list
  secrets/                 Secret management
  configs/                 Config management
  logs/                    Service log streaming
  contexts/                Docker context switcher
  help/                    Keybinding cheat sheet
  inspect/                 JSON inspect viewer
  networks/                Network list
  loading/                 Loading spinner
  commandinput/            ":" command bar
  searchinput/             "/" search filter bar (app-level, drives Filterable views)
  confirmdialog/           Confirmation prompts
  scaledialog/             Scale replica input
  helpbar/                 Dynamic keybinding bar
  systeminfo/              Header with cluster info
  viewstack/               Navigation stack (push/pop)
commands/
  api/                     Command context & arg parsing
  command/                 Top-level built-in commands (help.go, contexts.go, quit.go, alias.go, bootstrap.go, devupdate.go); docker-entity commands live under command/docker/<entity>/ls.go (service, node, network, volume, secret, config). devupdate.go registers `:dev-update` only when SWARMCLI_ENV=dev (force-shows the update-available notice for previewing)
  autoload.go              Blank import triggers init() registration
docker/
  client.go                Context-aware Docker client factory
  snapshot.go              In-memory cache (3s TTL, sync.RWMutex, atomic refresh flag)
  events.go                Docker event stream subscription
  service.go               Service ops: scale, restart, update
  node.go, task.go         Entity queries (TaskEntry includes ContainerID, populated from task.Status.ContainerStatus.ContainerID with nil-guard)
  stack.go                 Stack queries
  secret.go, config.go     Secret/config CRUD
registry/
  registry.go              Global command map: Register(), Get(), All(), Suggest()
utils/log/
  logger.go                zap wrapper: Init(), L(), Sync(), SetLevel(), lumberjack rotation
  slogcore.go              InitSlog(slog.Handler): forwards this package's output into a host program's log/slog handler instead of a file. For consumers importing swarmcli as a library (swarmcli-cd), where Init's lumberjack file is wrong and not initialising at all silently discards everything L()'s callers write
```

## Key Patterns

- **Bubble Tea MVC**: Input → Update() → tea.Cmd → View(). All state changes via `tea.Msg` types.
- **View Stack**: `viewStack.Push(old)` / `Pop()` for breadcrumb navigation.
- **View Factory**: Views auto-register via `init()` + `view.RegisterView(name, factory)` in each view's `register.go` (registry in `views/view/registry.go`, looked up with `GetFactory`). `app/app.go`'s blank import `_ "github.com/Eldara-Tech/swarmcli/views"` pulls `views/autoload.go`, which blank-imports every view — exactly mirroring the command autoload pattern.
- **Command Registry**: Commands in `commands/command/` auto-register via `init()` + `registry.Register()`. Accessed via `:` input.
- **Command Spec**: Commands optionally implement `registry.CommandWithSpec` (`Spec() registry.CommandSpec`, discovered by type assertion like `Aliaser`). The spec declares `Usage`, `Flags` (the allow-list), and `Examples`. `api.ParseInput` is the single chokepoint that, in order: short-circuits `Passthrough` specs, intercepts `--help`/`-h`/`-help` (and `:help <cmd>`) into a per-command help screen reusing the detailed help view, then rejects any undeclared flag (**global strict**, with a `did you mean --x?` suggestion). Unknown-flag rejection means every registered command MUST declare a spec — a missing/empty spec rejects all flags. `Passthrough:true` is the narrow escape-hatch for delegating/unavailable stubs (e.g. the OSS `bootstrap` stub): it skips both help interception and validation so every arg reaches `Execute` unchanged and the command keeps its own messaging (and no Pro flag internals leak into OSS — see Pro Feature Boundary).
- **Snapshot Cache**: `docker.GetSnapshot()` / `docker.RefreshSnapshot()` — 3s TTL, background event-driven invalidation.
- **Navigation**: `view.NavigateToMsg{ViewName, Payload, Replace}` dispatched in `update.go`.

## Adding New Functionality

**New command**: Create `commands/command/mycommand.go`, implement `registry.Command` (Name/Description/Execute), call `registry.Register()` in `init()`. Also implement `Spec() registry.CommandSpec` — declare every flag the command reads (`a.Has`/`a.Get`) plus `Usage`/`Examples`, or `:cmd --help` shows only a fallback and strict validation rejects the command's own flags. Aliases (`Aliaser`) inherit the primary's spec; do not add a spec to the alias. See `commands/command/docker/node/ls.go` for a zero-flag spec and `swarmcli-be/commands/pro/bootstrap.go` for the full worked example.

**New view**: Create `views/myview/`, implement `view.View` interface, and add a `register.go` whose `init()` calls `view.RegisterView(name, factory)`. Add its blank import to `views/autoload.go` so the package is loaded. See `views/nodes/register.go`.

**New `swarmcli charts <sub>` CLI subcommand** (this is *not* the TUI `:` registry — it's the arg-based dispatcher): add a `case` to the switch in `cli/charts.go` `chartsMain`, put the logic in `charts/` and keep the `cli/` half thin (the `charts` package is where the coverage bar applies). Any new flag goes in the single **global** `flags` struct in `cli/args.go` — note that every subcommand therefore parses every flag and *silently ignores* what it does not read, so if a flag would be a lie for your subcommand, reject it explicitly (see `rejectUnsupported` in `cli/apply.go`). Then update **all three** places the command list is duplicated, or they drift: `chartsUsage` in `cli/charts.go`, `README.md`, and `charts/README.md`.

## Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `SWARMCLI_ENV` | `dev` (console logs) or `prod` (JSON logs); also registers the dev-only `:dev-update` command (force-shows the update-available notice for previewing) | `prod` |
| `LOG_LEVEL` | `debug`/`info`/`warn`/`error` | `debug` (dev), `info` (prod) |
| `DOCKER_CONTEXT` | Override Docker context | `docker context show` |
| `TEST_LOG` | Enable logging in tests | unset |
| `SWARMCLI_CHARTS_ALLOW_PLAINTEXT` | Allow chart repositories served over plain `http://`; read only by `cli`, which wires it to `charts.RepoStore.AllowPlaintext` (the `charts` package never reads the environment, so embedders keep the https-only default) | unset (https only) |

## Pro Feature Boundary

The OSS repo must not contain pro implementation details — no pro-specific logic, no descriptions of how pro features work internally. Generic extension points (registries, hooks, feature flags) are fine; naming specific pro features or describing their internals is not. When adding code that will be called by pro, keep it generic and document it as an extension point without referencing pro specifics.

## Integration Test Infrastructure

- Tests in `integration-tests/` use `//go:build integration` tag
- `test-setup/docker-compose.yml`: DinD multi-node Swarm (1 manager on tcp://localhost:22375, 2 workers)
- `test-setup/test-stack.yml`: Demo services (whoami, whoami_single, log_ticker) with volumes, networks, and configs
- `test-setup/testenv.sh`: Orchestrator script
- Tests use `gotestsum` as test runner (with `--format=testname` locally, `--format=github-actions` in CI)
- Docker context name for tests: `swarmcli`
- When adding new resource types (volumes, networks, secrets, configs), update `test-setup/test-stack.yml` and add integration test assertions to ensure inspect and compose reconstruction cover them

## Pull Requests

Every PR to `main` must pass the `check_labels.yml` workflow which requires one label from each of three groups:

| Group | Labels | Meaning |
|---|---|---|
| A — Change type | `A0-ui`, `A1-feature`, `A2-bugfix`, `A3-technical` | What kind of change |
| B — Urgency | `B0-low-priority`, `B2-high-priority` | How urgent |
| C — Breaking | `C0-breaks-nothing`, `C1-breaking-change` | Backward compatibility |

Add all three labels when creating a PR: `gh pr edit <number> --add-label "A0-ui,B0-low-priority,C0-breaks-nothing"` (or use the REST API if `gh pr edit` fails due to classic projects deprecation).

When a PR fixes a GitHub issue, copy the issue's labels to the PR and add any missing required group labels (A, B, C). Use `gh api repos/OWNER/REPO/issues/<pr-number>/labels -f "labels[]=LABEL"` to add labels via API.

**Versioning at release.** A `C1-breaking-change` PR merged since the last GA forces a **major** tag (`vX.0.0`). The release changelog is type-only (no dedicated "Breaking" section), so the label is the breaking-change gate — the pushed tag is authoritative, overriding release-drafter's `$RESOLVED_VERSION`. For a breaking release, fill `.github/UPGRADE_NOTES.md` before tagging.

## CI Workflows (.github/workflows/)

- `ci.yml`: go fmt, golangci-lint, go build, Docker image build
- `integration-tests.yml`: Full E2E
- `release.yml`: GoReleaser on tags (multi-platform, Homebrew tap)
- `check_labels.yml`: PR label validation
- `licence.yml`: License header check

## Go Version & Build

Go 1.26. No Makefile — use `go build` directly. GoReleaser handles releases with `-trimpath -s -w` ldflags and version injection.

When updating the Go version, keep these three in sync:
- `go.mod` — `go` and `toolchain` directives
- `.devcontainer/Dockerfile` — `mcr.microsoft.com/devcontainers/go` image tag (tracks major.minor; patch versions are handled by `GOTOOLCHAIN=auto`)
- `govulncheck` CI step — bump suppressed vuln IDs if the new toolchain resolves them, or add new ones if it introduces new unfixed stdlib vulns
