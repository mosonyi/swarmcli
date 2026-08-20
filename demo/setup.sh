#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech
#
# Puts a cluster into the state the demo tapes record.
#
#   demo/setup.sh dind    3-node DinD swarm + the shop stack + three chart releases
#   demo/setup.sh stats   the container-statistics fixture, on a REAL daemon
#
# SWARMCLI_BIN overrides the binary used for `charts apply` (default: swarmcli).

set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
SWARMCLI_BIN="${SWARMCLI_BIN:-swarmcli}"
CONTEXT="swarm-demo"
MANAGER_HOST="tcp://localhost:22375"

info() { printf '\033[36m[demo]\033[0m %s\n' "$*"; }
die()  { printf '\033[31m[demo]\033[0m %s\n' "$*" >&2; exit 1; }

dind() {
  info "Starting the 3-node swarm with demo hostnames..."
  docker compose -f "$REPO/test-setup/docker-compose.yml" \
                 -f "$REPO/demo/docker-compose.demo.yml" up -d

  info "Waiting for the manager API on $MANAGER_HOST..."
  for _ in $(seq 1 30); do
    docker -H "$MANAGER_HOST" info >/dev/null 2>&1 && break
    sleep 2
  done
  docker -H "$MANAGER_HOST" info >/dev/null 2>&1 || die "manager never came up"

  # The context name is on screen in the TUI header, so it is part of the shot.
  docker context inspect "$CONTEXT" >/dev/null 2>&1 \
    || docker context create "$CONTEXT" --docker "host=$MANAGER_HOST" >/dev/null

  info "Waiting for both workers to join..."
  for _ in $(seq 1 60); do
    [ "$(docker --context "$CONTEXT" node ls --format '{{.Status}}' 2>/dev/null | grep -c Ready)" -ge 3 ] && break
    sleep 2
  done

  # The redis chart declares an external secret; pre-create it so `charts apply`
  # pre-flights clean. Never opened on camera.
  docker --context "$CONTEXT" secret inspect redis_password >/dev/null 2>&1 \
    || printf 'demo-not-a-real-password' | docker --context "$CONTEXT" secret create redis_password - >/dev/null

  info "Deploying the shop stack..."
  docker --context "$CONTEXT" stack deploy -c "$REPO/demo/stack-shop.yml" shop

  info "Installing the chart releases..."
  DOCKER_CONTEXT="$CONTEXT" "$SWARMCLI_BIN" charts apply -f "$REPO/demo/swarmcli-release.yaml"

  info "Ready. Record with: DOCKER_CONTEXT=$CONTEXT vhs demo/hero.tape"
}

stats() {
  # The one check that matters: a threaded cgroup hierarchy (DinD, some rootless
  # setups) refuses a memory limit and reports no MEM or BLK statistics at all,
  # which is exactly what this clip is meant to show.
  info "Checking that this daemon can enforce a memory limit..."
  docker run --rm --memory 64m alpine:3.20 true >/dev/null 2>&1 \
    || die "this daemon refuses memory limits — MEM and BLK would read 'not reported by this host'. Use a real dockerd, not DinD."

  docker info --format '{{.Swarm.LocalNodeState}}' | grep -q active \
    || { info "Initialising a single-node swarm..."; docker swarm init >/dev/null; }

  info "Deploying the stats fixture..."
  docker stack deploy -c "$REPO/demo/stack-stats.yml" stats

  cat <<'NEXT'

Next, by hand (they need the BE binary):
  1. swarmcli-be, then :bootstrap        — rbac-proxy + agent + agent-manager
  2. install a BE or trial licence       — :license
  3. leave it running ~2 minutes         — the agents fill their sample ring
  4. vhs demo/clip-stats.tape            — the tape opens on the 1-minute span
NEXT
}

case "${1:-}" in
  dind)  dind ;;
  stats) stats ;;
  *)     die "usage: demo/setup.sh {dind|stats}" ;;
esac
