# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech
#
# The hero beats, with no output or display settings of their own: hero.tape
# renders them as video and readme-gif.tape as a GIF, so the two never drift.
# Sourced, not run directly.
#
# Requires: demo/setup.sh dind, and DOCKER_CONTEXT=swarm-demo.

# Start the TUI off-camera and let the first snapshot land. The current asset's
# opening seconds show an empty CPU/MEM header because it started recording
# before the first system-info tick; this is that defect's fix.
Hide
Type "clear"
Enter
Type "swarmcli"
Enter
Sleep 8s
Show

# --- stacks: a real cluster, four stacks across three named nodes -----------
Sleep 4s
# 'shop' is the fourth row once the chart releases (edge, hello, sessions) are
# installed. Verify the row on the first dry run — the sort is the view's, not
# ours, and a wrong cursor here records the wrong stack.
Down 3
Sleep 1s

# --- services: drill into shop, then expand one service to its tasks --------
Enter
Sleep 3s
Down 1
Sleep 1s
Type "p"
Sleep 3s
Type "p"
Sleep 500ms

# --- logs: lines arriving live ---------------------------------------------
Type "l"
Sleep 6s
Escape
Sleep 1s
Escape
Sleep 1s

# --- charts: releases, with an upgrade offered in LATEST --------------------
Type ":charts"
Enter
Sleep 5s

# --- back to where we started, so the loop seam is invisible ----------------
Escape
Sleep 2s
