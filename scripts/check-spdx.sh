#!/usr/bin/env sh
# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech
set -eu

tmpfile=$(mktemp)
trap 'rm -f "$tmpfile"' EXIT

find . -type f \( -name '*.go' -o -name '*.sh' \) \
  -not -path './vendor/*' \
  -not -path './.git/*' \
  > "$tmpfile"

fail=0
while IFS= read -r f; do
  if ! head -n 20 "$f" | grep -q "SPDX-License-Identifier: Apache-2.0"; then
    echo "Missing SPDX header: $f"
    fail=1
  fi
done < "$tmpfile"

exit "$fail"
