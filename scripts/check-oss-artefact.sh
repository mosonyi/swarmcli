#!/usr/bin/env sh
# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech

set -eu

# The published swarmcli-oss artefact carries nothing private.
#
# One tag produces two artefacts: `swarmcli`, built from the private
# swarmcli-be wrapper and carrying the licensed code inert, and `swarmcli-oss`,
# built from this tree alone. The second one only means anything if it is
# verifiably the second one — "the released binary is not open source" has a
# rebuttal only while the rebuttal is checkable, and a build published under the
# wrong name is not a thing anyone would notice by reading it.
#
# So this reads the module graph Go stamped into the binary itself, rather than
# the graph the source implies. `go version -m` reports the main module and
# every module linked into it, which is the same information a `go.mod` would
# give and is the information that survives having been built somewhere else.
#
# Usage: ./scripts/check-oss-artefact.sh path/to/binary
#
# Run per artefact by .goreleaser.yml's post-build hook, so it sees each of the
# published binaries and not a locally built stand-in — and by ci.yml on every
# change, so it is not a gate that only ever runs on a tag.

if [ $# -ne 1 ]; then
  echo "usage: $0 <binary>" >&2
  exit 2
fi
binary=$1

# The major suffix is part of the path, not decoration: `go version -m` stamps
# the module line exactly as go.mod declares it, so a v1 value here refuses
# every binary this repository now builds.
main=github.com/Eldara-Tech/swarmcli/v2

# The private modules. Matched as whole module paths or as a parent of one, so
# that `…/swarmcli` is not read as a prefix of `…/swarmcli-be` in either
# direction — this repository's own module would otherwise match every entry.
# Listed without a major suffix on purpose: the parent match already covers
# `…/swarmcli-be/v2` and every major after it, so a private module going v3
# does not need an edit here to stay caught.
private='github.com/Eldara-Tech/swarmcli-be github.com/Eldara-Tech/swarmcli-cd-be'

stamp=$(go version -m "$binary")

# The `mod` line is the main module: what was built, as opposed to what it
# depends on. A merged binary published under the OSS name fails here first and
# most clearly.
got_main=$(echo "$stamp" | awk '$1 == "mod" { print $2; exit }')
if [ "$got_main" != "$main" ]; then
  echo "$binary was built from '$got_main', not from '$main'."
  echo "The swarmcli-oss artefact is this repository's own build; see docs/editions.md."
  exit 1
fi

fail=0
deps=$(echo "$stamp" | awk '$1 == "dep" || $1 == "=>" { print $2 }')

# Not zero, because a stamp this script cannot read would otherwise pass as a
# binary with no private dependencies — which is the shape of every guard that
# stops guarding.
if [ -z "$deps" ]; then
  echo "$binary reports no linked modules at all; this check read nothing and asserted nothing."
  exit 1
fi

for module in $private; do
  for dep in $deps; do
    case "$dep" in
    "$module" | "$module"/*)
      echo "$binary links $dep, which is not Apache-2.0 and is not this repository's."
      fail=1
      ;;
    esac
  done
done

if [ "$fail" -ne 0 ]; then
  echo
  echo "This artefact is published as the wholly Apache-2.0 one. Whatever pulled"
  echo "a private module in has to come out, or the artefact has to stop claiming"
  echo "to be that."
  exit 1
fi

echo "$binary: $main, $(echo "$deps" | wc -l | tr -d ' ') linked modules, none private."
