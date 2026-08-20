# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech
#
# The landing-page hero, as video. Run from the repo root:
#   DOCKER_CONTEXT=swarm-demo vhs demo/hero.tape
#
# 2400px wide is deliberate: the site renders the hero in a max-w-5xl (1024px)
# column, so anything under ~2048px is soft on a HiDPI display — which is the
# main visual complaint about the asset this replaces.

Output demo/out/hero.mp4
Output demo/out/hero.webm

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

Source demo/beats-hero.tape

# The poster the <video> element shows before playback — a real frame of the
# real product, replacing today's hand-drawn mock cluster.
Screenshot demo/out/hero-poster.png
