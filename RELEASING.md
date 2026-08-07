<!--
SPDX-License-Identifier: Apache-2.0
Copyright © 2026 Eldara Tech
-->

# Release process

Tagging is the whole trigger. `.github/workflows/release.yml` then drafts the
notes from PR labels, publishes binary archives with GoReleaser, updates the
Homebrew cask and the Scoop manifest, and pushes a multi-arch image to Docker
Hub. Nothing is run by hand.

**This repository publishes the `-oss` half of a release, not the whole of it.**
Each release carries two artefacts ([docs/editions.md](docs/editions.md)):

| Artefact | Tagged in | Publishes |
|---|---|---|
| `swarmcli_*`, `eldaratech/swarmcli:<tag>`, `:latest`, cask + manifest `swarmcli` | the private `swarmcli-be` wrapper | into **this** repository's releases, under `RELEASE_TOKEN` |
| `swarmcli-oss_*`, `eldaratech/swarmcli:<tag>-oss`, cask + manifest `swarmcli-oss` | here | into this repository's releases |

So a release needs **two tags, one in each repository, carrying the same version
string** — and the public one first.

- There is no trigger from this tag. A tag pushed here raises no event in
  `swarmcli-be`, so nothing starts the merged build but a tag pushed there;
  assuming otherwise produces a release that silently never built its default
  artefact.
- The order is not a preference. GoReleaser publishes into a release *named
  after the tag it was invoked with*, so this tag creating the release first is
  what gives the private run something to add to.
- Nothing collides, and that is by construction rather than by luck: the
  archives here are `swarmcli-oss_*`, the checksum file is `checksums-oss.txt`,
  and the image tag carries an `-oss` suffix. The merged pipeline writes
  `swarmcli_*`, `checksums-merged.txt` and the unsuffixed image tags, and it
  refuses to run if this repository at the pinned tag has gone back to the plain
  archive names.

**A tag pushed here on its own no longer moves `:latest`,** and publishes no
unsuffixed image tag at all. `:latest` is the merged artefact; two pipelines
competing for it would leave an operator with whichever finished last.

**`swarmcli-be` releases against a tag that must exist here.** Its release
refuses to publish unless this repository already has a published release for
the same version — including when this repository has no changes to ship, in
which case tag the same commit as the previous release and say so in the notes.
A BE-only patch (a compatibility-pin bump, say) still needs its counterpart
here.

## Prerequisites

Repository secrets, none of which the tooling can set for itself:

| Secret | |
|---|---|
| `DOCKERHUB_USERNAME` | Docker Hub account with push access to `eldaratech` |
| `DOCKERHUB_TOKEN` | an access token for it, not the password |
| `HOMEBREW_TAP_TOKEN` | `contents:write` on `Eldara-Tech/homebrew-tap` |
| `SCOOP_BUCKET_TOKEN` | `contents:write` on `Eldara-Tech/scoop-bucket` |

The GitHub release itself uses the job's own `GITHUB_TOKEN`.

## Before tagging

**Choose the version by hand.** release-drafter does not compute it — the
workflow passes the pushed tag, which overrides `$RESOLVED_VERSION`. If any PR
merged since the last GA carries `C1-breaking-change`, the tag MUST bump major:
the changelog is type-only, with no dedicated "Breaking" section, so the label is
the gate.

```bash
gh pr list --repo Eldara-Tech/swarmcli --state merged \
  --label C1-breaking-change --search "merged:>=<last-GA-date>"
```

**Fill `.github/UPGRADE_NOTES.md`** for a breaking release, before tagging. The
workflow prepends it as a "⚠️ Upgrade Notes" section at the top of the release
body; it ships as an HTML-comment placeholder, which is treated as empty. Clear
it back to the placeholder after the GA.

**Rehearse.** This is the whole of it, and it takes about three minutes:

```bash
goreleaser check
goreleaser release --snapshot --clean --skip=docker
```

Then read `dist/`: eleven `swarmcli-oss_*` archives plus `checksums-oss.txt`,
`dist/homebrew/Casks/swarmcli-oss.rb` and `dist/scoop/swarmcli-oss.json`. The
per-build hook runs `scripts/check-oss-artefact.sh` on every binary as it is
produced, so a snapshot that completes has already proved the artefact claim.

Confirm the executable inside an archive is still `swarmcli`, not
`swarmcli-oss` — the Homebrew cask's `binary`, Scoop's `bin`, swarmcli-charts'
`tar xzf … swarmcli` and every documented invocation depend on it:

```bash
tar tzf dist/swarmcli-oss_Linux_x86_64.tar.gz
```

Know what the rehearsal omits: `--snapshot` skips GoReleaser's git-state
validation entirely, so a dirty working tree passes the dry run and fails
against a pushed tag.

## Tagging

```bash
git tag -a v1.14.0 -m "Release v1.14.0"
git push originToken v1.14.0
```

Then tag `swarmcli-be` with the same string. Keep the gap short: between the two
the release has no default artefact and `:latest` still points at the previous
version.

Release candidates are tagged the same way (`v1.14.0-rc1`). GoReleaser marks
them prerelease automatically, and `skip_upload: auto` keeps an rc out of the
Homebrew tap and the Scoop bucket.

## Verifying

```bash
gh release view v1.14.0 --repo Eldara-Tech/swarmcli

# The OSS half, before the merged one lands:
#   11 archives + checksums-oss.txt, and no swarmcli_* asset yet.

curl -sSLO https://github.com/Eldara-Tech/swarmcli/releases/download/v1.14.0/swarmcli-oss_Linux_x86_64.tar.gz
tar xzf swarmcli-oss_Linux_x86_64.tar.gz && ./swarmcli version
# 1.14.0 (oss build, chart engine 1.14.0)

docker buildx imagetools inspect eldaratech/swarmcli:v1.14.0-oss
```

Both halves of that line are checks. `(oss build)` is stamped from the build
rather than from anything at runtime, so it is what tells the two artefacts
apart; a bare version string with no build marker means the ldflag did not take.

`chart engine unstamped` is a **failed release**, not a cosmetic problem. The
engine version is what `CheckCompat` compares a chart's declared
`swarmcliVersion` floor against, and an unstamped engine reports
`CompatUnknown` — so every chart deploys with its floor unchecked, and nothing
else about such a release looks wrong.

After the merged tag lands, verify the consumer rather than the log — query the
registry for every tag the release claims and compare digests, because
`:latest`, the version tag and the `-oss` tag are published by two different
pipelines and only one of them is in this repository.
