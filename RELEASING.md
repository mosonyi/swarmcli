# Release Process

This project uses [GoReleaser](https://goreleaser.com/) to build and publish releases across multiple platforms, similar to k9s.

## Supported Platforms

The release process builds binaries for:

- **Linux**: amd64, arm64, arm (v7), 386
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64, arm64, 386
- **FreeBSD**: amd64, arm64

## How to Create a Release

### 1. Install GoReleaser

```bash
# On macOS
brew install goreleaser

# On Linux
go install github.com/goreleaser/goreleaser/v2@latest
```

### 2. Create and push a tag

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

### 3. Run GoReleaser locally

```bash
# Export GitHub token (create one at https://github.com/settings/tokens)
export GITHUB_TOKEN="your_github_token"

# Run GoReleaser
goreleaser release --clean
```

This will:
- Build binaries for all platforms
- Create compressed archives (.tar.gz for Unix, .zip for Windows)
- Generate checksums
- Create a GitHub release with all artifacts
- Generate a changelog from commit messages

### 4. Find your release

Go to: https://github.com/mosonyi/swarmcli/releases

## Testing Locally

You can test the release process locally without publishing:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser/v2@latest

# Test the build (snapshot mode)
goreleaser release --snapshot --clean

# Check the dist/ folder for built binaries
ls -la dist/
```

## Version Information

Version information is embedded at build time using ldflags:
- `version`: The git tag (e.g., v0.1.0)
- `commit`: The git commit hash
- `date`: The build date

These can be accessed in the code via the variables in `main.go`.

## Changelog Format

The changelog is automatically generated from commit messages. For best results, use conventional commit format:

- `feat:` - New features
- `fix:` - Bug fixes
- `perf:` - Performance improvements
- `docs:`, `test:`, `ci:`, `chore:` - Excluded from changelog

Example:
```bash
git commit -m "feat: add horizontal scrolling to logs view"
git commit -m "fix: resolve alignment issue in nodes table"
```

## Optional Enhancements

The `.goreleaser.yml` includes commented sections for:

### Homebrew Tap
Automatically publish to a Homebrew tap for easy installation:
```bash
brew install mosonyi/tap/swarmcli
```

### Docker Images
Automatically build and push Docker images to Docker Hub or GitHub Container Registry.

Uncomment and configure these sections in `.goreleaser.yml` as needed.

## File Sizes

Release binaries are optimized with:
- `-trimpath`: Remove file system paths from binary
- `-s -w`: Strip debug information
- `CGO_ENABLED=0`: Static binaries with no external dependencies

This keeps binary sizes small and makes them portable across systems.
