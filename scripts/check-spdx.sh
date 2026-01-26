#!/usr/bin/env sh
set -eu

fail=0

check_file () {
  f="$1"
  if ! head -n 20 "$f" | grep -q "SPDX-License-Identifier: Apache-2.0"; then
    echo "Missing SPDX header: $f"
    fail=1
  fi
}

find . -type f \( -name '*.go' -o -name '*.sh' \) \
  -not -path './vendor/*' \
  -not -path './.git/*' \
  | while read -r f; do
      check_file "$f"
    done

exit "$fail"
