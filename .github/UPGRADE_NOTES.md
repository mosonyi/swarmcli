<!--
Upgrade Notes — curated breaking-change callout for the next release.

Leave this file as-is (this comment only) when the release has no breaking
changes. For a breaking release, replace the comment with migration prose in
Markdown; the release workflow prepends it as a "## ⚠️ Upgrade Notes" section
at the top of the GitHub release notes (see .github/workflows/release.yml).
Clear it back to this comment once the GA is cut.
-->

**The download named `swarmcli` is now the Business Edition build.** Nothing
about the executable changes — same name, same flags, same behaviour with no
licence — but the artefact you get from the unsuffixed name now contains
licensed code that is inert until a licence verifies. The wholly Apache-2.0
build is published beside it as `swarmcli-oss`.

| Was | Is now | The Apache-2.0 build is |
|---|---|---|
| `swarmcli_Linux_x86_64.tar.gz` | Business Edition build | `swarmcli-oss_Linux_x86_64.tar.gz` |
| `checksums.txt` | `checksums-merged.txt` (BE) | `checksums-oss.txt` |
| `eldaratech/swarmcli:v1.14.0`, `:latest` | Business Edition build | `eldaratech/swarmcli:v1.14.0-oss` |
| `brew install swarmcli` | Business Edition build | `brew install Eldara-Tech/tap/swarmcli-oss` |
| `scoop install swarmcli` | Business Edition build | `scoop install swarmcli-oss` |

**`brew upgrade swarmcli` and `scoop update swarmcli` will move you onto the
Business Edition build.** Neither package manager can express this as a rename,
so this note is the only place it can be said. If you want to stay on the
Apache-2.0 build, switch to `swarmcli-oss`. If you do nothing, the CLI keeps
working exactly as it does today — the licensed half stays inert and the TUI
still reports Community Edition.

**Scripts that download a release asset by name need updating**, whichever build
they want: the plain names now resolve to a different artefact and
`checksums.txt` no longer exists. Anything verifying a download must move to
`checksums-oss.txt` or `checksums-merged.txt`, which are separate files because
a release now carries two sets of artefacts and one manifest could not say which
set it covered.

**A tag on this repository alone no longer moves `:latest`.** The unsuffixed
image tags belong to the merged build, which is published by a second pipeline
into this same release.

`swarmcli version` now names which artefact you are running, and
[docs/editions.md](../docs/editions.md) is the full picture.
