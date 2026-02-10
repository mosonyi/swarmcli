# Contributing to SwarmCLI

We love your input! We want to make contributing to SwarmCLI as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Setup

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
