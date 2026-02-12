#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright © 2026 Eldara Tech

set -euo pipefail

# === Config ================================================================
COMPOSE_FILE="test-setup/docker-compose.yml"
MANAGER_HOST="tcp://localhost:22375"
KEEP="${KEEP:-0}"
FOLLOW="${FOLLOW:-0}"
SERVICE="${SERVICE:-}"
DOCKER_COMPOSE="docker compose -f $COMPOSE_FILE"
CONTEXT_NAME="swarmcli"

# === Colors ================================================================
RESET="\033[0m"
BOLD="\033[1m"
DIM="\033[2m"
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
MAGENTA="\033[35m"
CYAN="\033[36m"

timestamp() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# === Helpers ===============================================================
info() { echo -e "${CYAN}[$(timestamp)] [INFO]${RESET} $*"; }
ok()   { echo -e "${GREEN}[$(timestamp)] [OK]${RESET}   $*"; }
warn() { echo -e "${YELLOW}[$(timestamp)] [WARN]${RESET} $*"; }
err()  { echo -e "${RED}[$(timestamp)] [ERR]${RESET}  $*" >&2; }

run_or_warn() { "$@" || warn "Command failed: $*"; }

# Wait until the manager DinD exposes its Docker API on tcp://localhost:22375
wait_for_manager() {
  info "⏳ Waiting for DinD manager to be ready on ${MANAGER_HOST}..."
  local retries=30
  local wait_sec=2
  local i
  for i in $(seq 1 $retries); do
    if curl -fsS "${MANAGER_HOST}" >/dev/null 2>&1 || docker -H "$MANAGER_HOST" info >/dev/null 2>&1; then
      ok "Manager is ready!"
      return
    fi
    sleep "$wait_sec"
  done
  err "Manager did not become ready after $((retries * wait_sec)) seconds."
  exit 1
}

# Ensure the context exists and points to a live daemon
ensure_context() {
  wait_for_manager
  if docker context inspect "$CONTEXT_NAME" >/dev/null 2>&1; then
    info "Checking if context '$CONTEXT_NAME' is alive..."
    if ! docker --context "$CONTEXT_NAME" info >/dev/null 2>&1; then
      warn "Context '$CONTEXT_NAME' points to a non-running daemon. Recreating..."
      docker context rm -f "$CONTEXT_NAME"
      docker context create "$CONTEXT_NAME" --docker "host=$MANAGER_HOST"
    fi
  else
    info "Creating Docker context '$CONTEXT_NAME'..."
    docker context create "$CONTEXT_NAME" --docker "host=$MANAGER_HOST"
  fi
}

# === Before cmd_up() =======================================================
cleanup_port() {
  # Remove any old container using 22375 to avoid "port already allocated"
  if docker ps -q --filter "publish=22375" | grep . >/dev/null; then
    info "🧹 Removing old container(s) using port 22375..."
    docker ps -q --filter "publish=22375" | xargs -r docker rm -f
    sleep 2
  fi
}

# === Commands ==============================================================

cmd_up() {
  cleanup_port  # <<< ensure old manager gone

  info "🚀 Starting multinode Swarm environment..."
  $DOCKER_COMPOSE up -d
  $DOCKER_COMPOSE ps

  info "🔧 Ensuring Docker context..."
  ensure_context

  ok "Swarm multinode environment is up."
}

cmd_deploy() {
  info "📦 Deploying test stack..."
  docker --context "$CONTEXT_NAME" stack deploy -c test-setup/test-stack.yml demo
  info "⏳ Waiting for services to start..."
  sleep 20
  docker --context "$CONTEXT_NAME" stack ls
  docker --context "$CONTEXT_NAME" service ls
  ok "Test stack deployed successfully."
}

cmd_test() {
  info "🧪 Running Go integration tests..."

  local test_name="${1:-}"   # optional single test
  local format="testname"    # default local format

  # CI-specific settings
  local junit_file=""
  if [[ "${CI:-0}" -eq 1 ]]; then
    format="github-actions"
    junit_file="/tmp/test-report.xml"
  elif [[ "${VERBOSE:-0}" -eq 1 ]]; then
    format="standard-verbose"
  fi

  local args=("--format=$format")

  # Add JUnit file only if set
  [[ -n "$junit_file" ]] && args+=("--junitfile=$junit_file")

  # Always include integration build tag
  local go_test_cmd=("-tags=integration" "-v")

  if [[ -n "$test_name" ]]; then
    info "🎯 Running single test: $test_name"
    go_test_cmd+=("-run" "$test_name" "./integration-tests/...")
  else
    info "🧩 Running all integration tests"
    go_test_cmd+=("./integration-tests/...")
  fi

  # Combine into gotestsum command
  local full_cmd=("gotestsum" "${args[@]}" "--" "${go_test_cmd[@]}")

  # Print full command for transparency
  echo -e "\n📜 Executing command:\n${full_cmd[*]}\n"

  # Run gotestsum using the Docker context
  DOCKER_CONTEXT="$CONTEXT_NAME" "${full_cmd[@]}"
}

cmd_down() {
  info "🧹 Tearing down Swarm environment..."

  # Ensure context exists
  if ! docker context inspect "$CONTEXT_NAME" >/dev/null 2>&1; then
    info "Creating Docker context '$CONTEXT_NAME' for teardown..."
    docker context create "$CONTEXT_NAME" --docker "host=$MANAGER_HOST"
  fi

  # Remove the stack (ignore errors if it doesn't exist)
  run_or_warn docker --context "$CONTEXT_NAME" stack rm demo

  # Bring down compose services and volumes
  run_or_warn $DOCKER_COMPOSE down -v

  # Remove stopped containers
  info "🗑 Removing all stopped containers..."
  docker container prune -f || warn "Failed to prune containers"

  # Remove unused volumes
  info "🗑 Removing unused volumes..."
  docker volume prune -f || warn "Failed to prune volumes"

  # Remove unused networks
  info "🗑 Removing unused networks..."
  docker network prune -f || warn "Failed to prune networks"

  # Remove dangling images
  info "🗑 Removing dangling images..."
  docker image prune -f || warn "Failed to prune images"

  ok "Swarm environment and unused resources cleaned up."
}

cmd_clean() {
  info "🧼 Cleaning up contexts and resources..."
  run_or_warn docker context rm -f "$CONTEXT_NAME" node1 node2
  ok "Clean up complete."
}

cmd_integration() {
  cmd_up
  cmd_deploy
  cmd_test

  if [[ "$KEEP" -eq 1 ]]; then
    warn "KEEP=1 set — leaving environment running for inspection."
  else
    cmd_down
    cmd_clean
  fi
}

# === Dispatcher ============================================================
case "${1:-}" in
  up|deploy|logs|down|clean|integration)
    cmd_"$1"
    ;;
  test)
    # Pass optional single test name to cmd_test
    cmd_test "${2:-}"
    ;;
  *)
    echo -e "${BOLD}Usage:${RESET} $0 {up|deploy|test|logs|down|clean|integration} [test_name]"
    exit 1
    ;;
esac
