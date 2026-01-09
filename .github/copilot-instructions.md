# Copilot / Agent Instructions for swarmcli

This document gives focused, actionable knowledge to help an AI coding agent be productive in this repo.

High-level architecture
- **CLI app entry:** [main.go](main.go) creates a Bubble Tea `Program` and calls `app.Init()`.
- **App bootstrap & view registry:** [app/app.go](app/app.go) initializes logging, command autoload (`import "swarmcli/commands"`), and registers views via `registerView(name, factory)`. Views are built with the `view.Factory` pattern.
- **Views & UI:** UI components live under `views/` and `ui/`. Each view is a Bubble Tea model; common patterns: `Init()` returns a `tea.Cmd`, views expose `SetSize`, `View()`, and handle messages in `update.go`.
- **Docker integration:** The `docker/` package wraps Docker CLI + SDK. `docker.GetClient()` respects `DOCKER_CONTEXT` (or `docker context show`) and TLS cert layout. See [docker/client.go](docker/client.go) and [docker/context.go](docker/context.go).
- **Commands registry:** The `commands` package uses autoloading to register CLI commands with `registry`. See [app/app.go](app/app.go) and `commands/` for patterns.

Key developer workflows
- Run locally (dev): `go run .` — development logs (pretty) are enabled with `SWARMCLI_ENV=dev go run .`.
- Build/run in container: `docker build -t swarmcli-dev .` then `docker run --rm -it -v "$PWD":/app -v /var/run/docker.sock:/var/run/docker.sock -w /app swarmcli-dev`.
- Tests / integration tests: Integration tests are driven by `test-setup/testenv.sh`. Use `TEST_LOG=1 ./test-setup/testenv.sh test` to run tests with visible logs. CI uses `bash test-setup/testenv.sh up && bash test-setup/testenv.sh deploy` then `bash test-setup/testenv.sh test` (see .github/workflows/integration-tests.yml).
- Docker contexts for tests: The project creates a Docker context pointing at a local DinD manager; `DOCKER_CONTEXT` environment variable is honored by `docker.GetClient()` and tests (see `integration-tests/*` and `test-setup/testenv.sh`).

Project-specific conventions & patterns
- Bubble Tea / view factory: Views are registered centrally in `app.Init()` and must return `(view.View, tea.Cmd)`. Prefer returning a ready-to-run `tea.Cmd` (often `model.Init()` or `tea.Batch(...)`). Example registration: services, nodes, contexts in [app/app.go](app/app.go).
- UI composition: Shared UI helpers are under `ui/` (framed boxes, overlays, status bar). Prefer these helpers for consistent look/feel (examples in `views/configs/view.go` and `views/stacks/*`).
- Filterable lists: Use the `ui/components/filterable` package for lists (cursor, search modes). Items implement `FilterValue()`, `Title()`, `Description()` — see `views/configs/view.go` for concrete types.
- Error & dialogs: Reuse `ui/components/errordialog` and `ui.RenderConfirmDialog` for modal flows; views typically toggle `...DialogActive` booleans and overlay rendered content with `ui.OverlayCentered()`.
- CI / logging: Logging initialized in `app.Init()` (via `utils/log`). Default logs go to `~/.local/state/swarmcli/app.log`. Tests and CI may set `TEST_LOG=1` to change behavior (see `utils/log/logger.go`).

Integration points & external dependencies
- Docker CLI + Docker SDK: The app uses both `exec.Command("docker", ...)` and `github.com/docker/docker/client`. Be careful with contexts and TLS cert paths (see `docker/client.go`).
- Test harness: `test-setup` uses Docker Compose to spin up a small Swarm cluster (DinD). The integration test harness relies on `docker context create` and `docker --context ... stack deploy` flow.

Helpful code pointers (quick examples)
- Registering views: [app/app.go](app/app.go) — follow its `registerView` usage.
- Docker client: [docker/client.go](docker/client.go) — honor `DOCKER_CONTEXT` and TLS cert layout; use `GetClient()` for network calls.
- Filterable item example: `views/configs/view.go` defines `configItem` (methods `FilterValue`, `Title`, `Description`).
- Test runner: `test-setup/testenv.sh` and `integration-tests/` — run the full integration environment from the repo root.

Coding agent DOs and DON'Ts (repo-specific)
- DO: Keep changes small and focused; many UI patterns rely on consistent rendering sizes and overlay behavior. Prefer using existing `ui` helpers.
- DO: Use `registry` and existing command constructors when adding commands so `app.Init()` picks them up via the autoload import in [app/app.go](app/app.go).
- DO: Use `docker.GetClient()` and `docker` helpers rather than invoking Docker directly unless adding a CLI helper (which should live in `docker/` package itself).
- DO NOT: Modify global view registration patterns or change the `view.Factory` signature — it's central to UI bootstrapping.
- DO NOT: Change logging paths or env var names without updating `app.Init()` and `utils/log/logger.go` and mentioning the impact on tests.

If unsure, inspect these files first
- [app/app.go](app/app.go)
- [main.go](main.go)
- [docker/client.go](docker/client.go)
- [test-setup/testenv.sh](test-setup/testenv.sh)
- [views/configs/view.go](views/configs/view.go)
- [utils/log/logger.go](utils/log/logger.go)
