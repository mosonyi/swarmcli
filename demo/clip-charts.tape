# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech
#
# The chart engine, on the DinD swarm prepared by `demo/setup.sh dind`.
#   DOCKER_CONTEXT=swarm-demo vhs demo/clip-charts.tape

Output demo/out/clip-charts.mp4
Output demo/out/clip-charts.webm

Require swarmcli

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
Type "swarmcli"
Enter
Sleep 8s
Show

Type ":charts"
Enter
Sleep 5s

# Two releases offer a newer chart in LATEST, one reads ✓.
Down 1
Sleep 2s

# The rendered manifest, then the stored values.
Type "i"
Sleep 4s
Escape
Sleep 1s
Type "v"
Sleep 4s
Escape
Sleep 2s
