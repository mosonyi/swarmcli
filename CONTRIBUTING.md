# Contributing to SwarmCLI

We love your input! We want to make contributing to SwarmCLI as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Setup

Before changing anything non-trivial, read
[docs/architecture.md](docs/architecture.md) — it maps the packages, explains how
views and commands self-register through `init()`, and names the seams the
Business Edition attaches to (renaming one of those is a cross-repo change).

SwarmCLI is built in **Go**. To get started:

1. **Clone the repository**:

   ```bash
   git clone https://github.com/Eldara-Tech/swarmcli.git
   cd swarmcli
   ```

2. **Run locally**:

   ```bash
   go run .
   ```

3. **Debug logging**:
   ```bash
   SWARMCLI_ENV=dev LOG_LEVEL=debug go run .
   ```

## Testing the TUI

Two assertion mistakes recur here, both of which produce a green test that never
reached the code it was written for.

**Call the real factory, do not model it.** `app.switchToView` batches the *view
factory's returned command* with `OnEnter()`. PR #574 gave eight views a test
asserting that entry arms exactly one poll chain, and modelled the factory as
`tea.Batch(m.Init(), m.OnEnter())`. Seven factories really did just return a load
command, so the test was right about those — but `views/services/register.go`
returns `tickCmd()` and `views/tasks/register.go` returns `model.OnEnter()`
outright. Both double-armed, and both stayed green **in the very PR whose subject
was that defect**. Calling `factory(m.deps, 80, 24, nil)` and putting *its*
command in the batch fails on both.

A hand-written stand-in encodes what you believe the caller does, so it can only
fail where your belief was already right — and the views that break the pattern
are exactly the ones the guess omits.

**Count the rows that carry content, not the height.** When a layout pads to a
fixed height, `require.Equal(t, height, lipgloss.Height(out))` passes whether the
space is *used* or *wasted*. Writing the guard for swarmcli#560, the fullscreen
frame trimmed and padded to exactly the terminal height, so a view sized three
rows too small still rendered full-height and the mutation that undersized it
stayed green.

The property under test was never "how tall is the output" — it was "did the
layout claim every row it was given". Give the fixture more content than can fit,
then assert on the rows that carry content.

**Mutation-check any guard you add**: break the thing it protects and confirm the
test fails. A guard that has never failed is an untested claim.

## Pull Request Process

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.

### Specific Commands

Please use specific commands instead of greedy commands to avoid adding unwanted files:

```bash
git add <file1> <file2>
# Avoid: git add .
```

## Community

Join us in our [GitHub Issues](https://github.com/Eldara-Tech/swarmcli/issues) for discussions!
