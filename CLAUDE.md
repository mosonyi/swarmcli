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
main.go                    Entry point; version injection via ldflags, tea.NewProgram()
app/
  app.go                   View factory registry, Init(), command autoload via _ "swarmcli/commands"
  model.go                 Central state: Model struct (viewport, currentView, viewStack, commandInput, systemInfo)
  update.go                Main message router: navigation, resize, events, key dispatch
views/
  view/interface.go        View contract: Update/View/Init/Name/OnEnter/OnExit/HasErrors/ShortHelpItems
  stacks/                  Stack list → drill into services
  services/                Service list (filterable by stack/node/all), scale/restart actions
  tasks/                   Task list per service/stack
  nodes/                   Cluster node list
  secrets/                 Secret management + reveal
  configs/                 Config management
  logs/                    Service log streaming
  contexts/                Docker context switcher
  help/                    Keybinding cheat sheet
  inspect/                 JSON inspect viewer
  networks/                Network list
  loading/                 Loading spinner
  commandinput/            ":" command bar
  confirmdialog/           Confirmation prompts
  scaledialog/             Scale replica input
  revealsecret/            Secret reveal (temp container)
  helpbar/                 Dynamic keybinding bar
  systeminfo/              Header with cluster info
  viewstack/               Navigation stack (push/pop)
commands/
  api/                     Command context & arg parsing
  command/                 Built-in commands (help, contexts, stacks, services, etc.)
  autoload.go              Blank import triggers init() registration
docker/
  client.go                Context-aware Docker client factory
  snapshot.go              In-memory cache (3s TTL, sync.RWMutex, atomic refresh flag)
  events.go                Docker event stream subscription
  service.go               Service ops: scale, restart, update
  node.go, task.go         Entity queries
  stack.go                 Stack queries
  secret.go, config.go     Secret/config CRUD
registry/
  registry.go              Global command map: Register(), Get(), All(), Suggest()
utils/log/
  logger.go                zap wrapper: Init(), L(), Sync(), SetLevel(), lumberjack rotation
```

## Key Patterns

- **Bubble Tea MVC**: Input → Update() → tea.Cmd → View(). All state changes via `tea.Msg` types.
- **View Stack**: `viewStack.Push(old)` / `Pop()` for breadcrumb navigation.
- **View Factory**: `viewRegistry[name]` maps view names to constructor functions, registered in `app.Init()`.
- **Command Registry**: Commands in `commands/command/` auto-register via `init()` + `registry.Register()`. Accessed via `:` input.
- **Snapshot Cache**: `docker.GetSnapshot()` / `docker.RefreshSnapshot()` — 3s TTL, background event-driven invalidation.
- **Navigation**: `view.NavigateToMsg{ViewName, Payload, Replace}` dispatched in `update.go`.

## Adding New Functionality

**New command**: Create `commands/command/mycommand.go`, implement `registry.Command` (Name/Description/Execute), call `registry.Register()` in `init()`.

**New view**: Create `views/myview/`, implement `view.View` interface, register factory in `app/app.go` `Init()`.

## Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `SWARMCLI_ENV` | `dev` (console logs) or `prod` (JSON logs) | `prod` |
| `LOG_LEVEL` | `debug`/`info`/`warn`/`error` | `debug` (dev), `info` (prod) |
| `DOCKER_CONTEXT` | Override Docker context | `docker context show` |
| `TEST_LOG` | Enable logging in tests | unset |

## Integration Test Infrastructure

- Tests in `integration-tests/` use `//go:build integration` tag
- `test-setup/docker-compose.yml`: DinD multi-node Swarm (1 manager on tcp://localhost:22375, 2 workers)
- `test-setup/test-stack.yml`: Demo services (whoami, whoami_single, etc.)
- `test-setup/testenv.sh`: Orchestrator script
- Tests use `gotestsum` as test runner (with `--format=testname` locally, `--format=github-actions` in CI)
- Docker context name for tests: `swarmcli`

## CI Workflows (.github/workflows/)

- `ci.yml`: go fmt, golangci-lint, go build, Docker image build
- `integration-tests.yml`: Full E2E
- `release.yml`: GoReleaser on tags (multi-platform, Homebrew tap)
- `check_labels.yml`: PR label validation
- `licence.yml`: License header check

## Go Version & Build

Go 1.25. No Makefile — use `go build` directly. GoReleaser handles releases with `-trimpath -s -w` ldflags and version injection.
