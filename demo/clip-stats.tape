# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech
#
# Container statistics (Business Edition). Records on a REAL single-node daemon
# after `demo/setup.sh stats`, :bootstrap, a licence and ~2 minutes of warm-up —
# under DinD the MEM and BLK panes read "not reported by this host" and the
# clip is pointless. SWARMCLI_BIN must be the BE binary.

Output demo/out/clip-stats.mp4
Output demo/out/clip-stats.webm

Set Shell "bash"
Set FontSize 24
Set Width 2400
Set Height 1300
Set Padding 24
Set Framerate 30
Set TypingSpeed 60ms
Set CursorBlink false
Set WindowBar Colorful
Set Theme { "background": "#0b0b0d", "foreground": "#e6e6e6", "cursor": "#e6e6e6", "selection": "#2a3f6b" }

Hide
Type "clear"
Enter
Type "${SWARMCLI_BIN:-swarmcli-be}"
Enter
Sleep 8s
# Into the stats stack's services, off-camera, so the clip opens on the graphs.
Enter
Sleep 3s
Down 1
Sleep 1s
Show

# Four panes, with the memory pane scaled to the service's declared 128MB limit.
Type "t"
Sleep 6s

# The span cycles 1 → 5 → 15 minutes.
Type "w"
Sleep 4s

# Pause, so a spike can be read.
Type "p"
Sleep 3s
Type "p"
Sleep 2s
