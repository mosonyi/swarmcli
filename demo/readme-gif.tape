# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech
#
# The same beats as the hero, rendered as the GIF that swarmcli's README shows.
# Smaller and slower on purpose: a 2400px 30fps GIF is tens of megabytes, and
# the README asset it replaces is already 5.5MB. Target: under 3MB.
#
#   DOCKER_CONTEXT=swarm-demo vhs demo/readme-gif.tape

Output assets/swarmcli.gif

Require swarmcli

Set Shell "bash"
Set FontSize 16
Set Width 1400
Set Height 760
Set Padding 16
Set Framerate 20
Set TypingSpeed 60ms
Set CursorBlink false
Set WindowBar Colorful
Set Theme { "background": "#0b0b0d", "foreground": "#e6e6e6", "cursor": "#e6e6e6", "selection": "#2a3f6b" }

Source demo/beats-hero.tape
