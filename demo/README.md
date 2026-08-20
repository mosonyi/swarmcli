# demo/ — the recording fixture

Everything the landing-page hero and the feature clips are recorded from:
the workload on screen, the chart releases, and the [VHS](https://github.com/charmbracelet/vhs)
tapes that drive the TUI. It exists so the demo is *re-recordable* — the header
shows the version, so every release ages the asset, and the previous one was a
hand-timed capture nobody could reproduce.

Design and decisions: swarmcli-website issue #85.

## Two environments, and why

| Clip | Where | Why |
|---|---|---|
| hero, charts | the 3-node DinD swarm (`test-setup/`) | cheap, reproducible, and it puts a real multi-node cluster on screen |
| stats | a **real** single-node daemon | DinD's cgroup hierarchy is threaded: `memory` and `io` are domain controllers and cannot be enabled there, so the MEM and BLK panes read *"not reported by this host"*, and a service declaring a memory limit will not start at all |

`setup.sh stats` refuses to run on a daemon that cannot enforce a memory limit,
which is the same condition, checked directly.

## Recording

```bash
# hero + charts (3-node DinD)
demo/setup.sh dind
DOCKER_CONTEXT=swarm-demo vhs demo/hero.tape        # mp4 + webm + poster frame
DOCKER_CONTEXT=swarm-demo vhs demo/readme-gif.tape  # assets/swarmcli.gif
DOCKER_CONTEXT=swarm-demo vhs demo/clip-charts.tape

# stats (real daemon, Business Edition binary)
demo/setup.sh stats
# then :bootstrap, install a licence, wait ~2 minutes for the agents to sample
SWARMCLI_BIN=swarmcli-be vhs demo/clip-stats.tape
```

Needs `vhs`, `ttyd` and `ffmpeg` on the recording host. Output lands in
`demo/out/` (gitignored) except the README GIF, which VHS writes straight to
`assets/swarmcli.gif`.

`beats-hero.tape` holds the hero's keystrokes and nothing else; `hero.tape` and
`readme-gif.tape` both `Source` it with different geometry, so the video and the
GIF can never drift apart.

## Before shipping a recording

Skim it frame by frame against this list. Every item is a defect the previous
asset shipped with:

- [ ] No personal registry namespaces, host paths, host IPs, or `docker-desktop` node names.
- [ ] No third-party configuration on screen (the old one ended in `nano` on a real `haproxy.cfg`).
- [ ] No licence token or customer identity; `:license` stays out of the cut.
- [ ] Header CPU/MEM populated from the very first frame.
- [ ] Log timestamps recent and monotonic.
- [ ] Last frame ≡ first frame, so the loop seam is invisible.
