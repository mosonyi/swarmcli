#!/usr/bin/env bash
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

  local args=(-tags=integration ./integration-tests/...)
  local tmp_log=""
  tmp_log="$(mktemp || true)"
  trap '[[ -n "${tmp_log:-}" ]] && rm -f "$tmp_log" || true' EXIT

  local pass_count=0
  local fail_count=0

  # Run tests and parse structured output
  if ! DOCKER_CONTEXT="$CONTEXT_NAME" go test "${args[@]}" -v 2>&1 | tee "$tmp_log" | while IFS= read -r line; do
    case "$line" in
      ===\ RUN*)
        echo -e "${BLUE}[$(timestamp)] [TEST]${RESET}  ${line#=== RUN   }"
        ;;
      ---\ PASS:*)
        echo -e "${GREEN}[$(timestamp)] [OK]${RESET}    ${line#--- PASS: }"
        ((pass_count++))
        ;;
      ---\ FAIL:*)
        echo -e "${RED}[$(timestamp)] [ERR]${RESET}   ${line#--- FAIL: }"
        ((fail_count++))
        ;;
      ok*\ \(*s\))
        echo -e "${GREEN}[$(timestamp)] [PASS]${RESET}  ${line}"
        ;;
      FAIL*\ \(*s\))
        echo -e "${RED}[$(timestamp)] [FAIL]${RESET}  ${line}"
        ;;
      FAIL*)
        echo -e "${RED}[$(timestamp)] [FAIL]${RESET}  ${line}"
        ;;
    esac
  done; then
    # Exit code of `go test` propagates here because of `!`
    echo
    warn "Some tests failed. Collecting failure details..."
    echo

    # Print failed test names
    mapfile -t failed_tests < <(grep '^--- FAIL:' "$tmp_log" | sed 's/^--- FAIL: //; s/ (.*)//')

    for test_name in "${failed_tests[@]}"; do
      echo -e "${RED}[$(timestamp)] [FAIL]${RESET}  ${YELLOW}$test_name${RESET}"
      echo -e "        👉 To inspect logs manually: ${CYAN}SERVICE=demo_whoami ./test-setup/testenv.sh logs${RESET}"
      echo
      docker --context "$CONTEXT_NAME" service logs demo_whoami --no-task-ids --timestamps 2>/dev/null \
        | tail -n 30 | sed 's/^/'"$(timestamp) ${YELLOW}[STACK]${RESET}"' /'
      echo
    done

    fail_count=${#failed_tests[@]}
    pass_count=$(grep -c '^--- PASS:' "$tmp_log" || true)

    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${BOLD}📊 TEST SUMMARY${RESET}"
    echo -e "  ✅ Passed: ${GREEN}${pass_count}${RESET}"
    echo -e "  ❌ Failed: ${RED}${fail_count}${RESET}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo

    err "Integration tests failed."
    return 1
  fi

  pass_count=$(grep -c '^--- PASS:' "$tmp_log" || true)
  fail_count=$(grep -c '^--- FAIL:' "$tmp_log" || true)

  echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo -e "${BOLD}📊 TEST SUMMARY${RESET}"
  echo -e "  ✅ Passed: ${GREEN}${pass_count}${RESET}"
  echo -e "  ❌ Failed: ${RED}${fail_count}${RESET}"
  echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"

  if [[ "$fail_count" -eq 0 ]]; then
    ok "Integration tests completed successfully."
  else
    err "Integration tests completed with failures."
  fi
}

cmd_logs() {
  info "📜 Collecting logs..."

  if [[ -n "$SERVICE" ]]; then
    info "🎯 Filtering logs for service: ${CYAN}$SERVICE${RESET}"
    if [[ "$FOLLOW" == "1" ]]; then
      info "👀 Following logs..."
      docker --context swarmcli service logs "$SERVICE" --no-task-ids --timestamps -f 2>/dev/null \
        | sed 's/^/'"$(timestamp) ${CYAN}[SERVICE]${RESET}"' /'
    else
      info "Showing last 100 lines for service $SERVICE"
      docker --context swarmcli service logs "$SERVICE" --no-task-ids --timestamps 2>/dev/null \
        | tail -n 100 | sed 's/^/'"$(timestamp) ${CYAN}[SERVICE]${RESET}"' /'
    fi
  else
    if [[ "$FOLLOW" == "1" ]]; then
      info "👀 Streaming all logs live (Ctrl+C to stop)..."
      (
        $DOCKER_COMPOSE logs -f manager 2>/dev/null | sed 's/^/'"$(timestamp) ${GREEN}[MANAGER]${RESET}"' /' &
        $DOCKER_COMPOSE logs -f worker1 2>/dev/null | sed 's/^/'"$(timestamp) ${BLUE}[WORKER1]${RESET}"' /' &
        $DOCKER_COMPOSE logs -f worker2 2>/dev/null | sed 's/^/'"$(timestamp) ${MAGENTA}[WORKER2]${RESET}"' /' &
        docker --context swarmcli service logs demo_whoami --no-task-ids --timestamps -f 2>/dev/null \
          | sed 's/^/'"$(timestamp) ${YELLOW}[STACK]${RESET}"' /' &
        wait
      )
    else
      info "Showing last 100 lines of all logs..."
      echo -e "${GREEN}=== 🟩 Manager ===${RESET}"
      $DOCKER_COMPOSE logs --no-color manager | tail -n 100 | sed 's/^/'"$(timestamp) ${GREEN}[MANAGER]${RESET}"' /'
      echo -e "${BLUE}=== 🟦 Worker1 ===${RESET}"
      $DOCKER_COMPOSE logs --no-color worker1 | tail -n 100 | sed 's/^/'"$(timestamp) ${BLUE}[WORKER1]${RESET}"' /'
      echo -e "${MAGENTA}=== 🟪 Worker2 ===${RESET}"
      $DOCKER_COMPOSE logs --no-color worker2 | tail -n 100 | sed 's/^/'"$(timestamp) ${MAGENTA}[WORKER2]${RESET}"' /'
      echo -e "${YELLOW}=== 🟨 Swarm Services ===${RESET}"
      docker --context swarmcli service logs demo_whoami --no-task-ids --timestamps 2>/dev/null \
        | tail -n 100 | sed 's/^/'"$(timestamp) ${YELLOW}[STACK]${RESET}"' /'
    fi
  fi
}

cmd_down() {
  info "🧹 Tearing down Swarm environment..."
  # Re-create context if missing to avoid "context not found"
  if ! docker context inspect "$CONTEXT_NAME" >/dev/null 2>&1; then
    info "Creating Docker context '$CONTEXT_NAME' for teardown..."
    docker context create "$CONTEXT_NAME" --docker "host=$MANAGER_HOST"
  fi

  run_or_warn docker --context "$CONTEXT_NAME" stack rm demo
  run_or_warn $DOCKER_COMPOSE down -v
  ok "Swarm environment torn down."
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
  up|deploy|test|logs|down|clean|integration)
    cmd_"$1"
    ;;
  *)
    echo -e "${BOLD}Usage:${RESET} $0 {up|deploy|test|logs|down|clean|integration}"
    exit 1
    ;;
esac
