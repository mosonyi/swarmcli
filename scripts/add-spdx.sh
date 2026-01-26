#!/usr/bin/env sh
set -eu

YEAR=2026
OWNER="Eldara Tech"

add_go_header() {
  file="$1"
  if head -n 20 "$file" | grep -q "SPDX-License-Identifier"; then
    return
  fi

  tmp="$(mktemp)"
  {
    echo "// SPDX-License-Identifier: Apache-2.0"
    echo "// Copyright © $YEAR $OWNER"
    echo
    cat "$file"
  } > "$tmp"
  mv "$tmp" "$file"
  echo "Added SPDX header: $file"
}

add_sh_header() {
  file="$1"
  if head -n 20 "$file" | grep -q "SPDX-License-Identifier"; then
    return
  fi

  tmp="$(mktemp)"

  read -r first_line < "$file" || true
  if echo "$first_line" | grep -q '^#!'; then
    {
      echo "$first_line"
      echo "# SPDX-License-Identifier: Apache-2.0"
      echo "# Copyright © $YEAR $OWNER"
      echo
      tail -n +2 "$file"
    } > "$tmp"
  else
    {
      echo "# SPDX-License-Identifier: Apache-2.0"
      echo "# Copyright © $YEAR $OWNER"
      echo
      cat "$file"
    } > "$tmp"
  fi

  mv "$tmp" "$file"
  echo "Added SPDX header: $file"
}

# Go files
find . -type f -name '*.go' \
  -not -path './vendor/*' \
  -not -path './.git/*' \
  | while read -r f; do
      add_go_header "$f"
    done

# Shell scripts
find . -type f -name '*.sh' \
  -not -path './vendor/*' \
  -not -path './.git/*' \
  | while read -r f; do
      add_sh_header "$f"
    done
