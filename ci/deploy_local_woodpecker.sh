#!/usr/bin/env bash

set -euo pipefail

CI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${CI_DIR}/.env"
COMPOSE_FILE="${CI_DIR}/docker-compose.yml"

check_deps() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker is required but not installed."
    exit 1
  fi

  if ! docker compose version >/dev/null 2>&1; then
    echo "docker compose plugin is required but not installed."
    exit 1
  fi
}

write_env_if_missing() {
  if [[ -f "${ENV_FILE}" ]]; then
    return
  fi

  local secret
  secret="$(openssl rand -hex 32)"

  cat >"${ENV_FILE}" <<EOF
# Generate this once and keep it stable.
WOODPECKER_AGENT_SECRET=${secret}

# Local URLs.
WOODPECKER_HOST=http://localhost:8000
WOODPECKER_SERVER_ADDR=:8000
WOODPECKER_GRPC_ADDR=:9000

# SQLite for a local setup.
WOODPECKER_DATABASE_DRIVER=sqlite3
WOODPECKER_DATABASE_DATASOURCE=/var/lib/woodpecker/woodpecker.sqlite

# Allow initial user registration.
WOODPECKER_OPEN=true

# GitHub integration (fill values after creating a GitHub OAuth app).
WOODPECKER_GITHUB=true
WOODPECKER_GITHUB_CLIENT=replace-with-github-oauth-client-id
WOODPECKER_GITHUB_SECRET=replace-with-github-oauth-client-secret

# Shared secret and server endpoint used by the agent.
WOODPECKER_SERVER=woodpecker-server:9000
EOF

  echo "Created ${ENV_FILE}. Update OAuth values before logging in to Woodpecker."
}

start_stack() {
  docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up -d
}

stop_stack() {
  docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" down
}

show_logs() {
  docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" logs -f
}

status() {
  docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" ps
}

usage() {
  echo "Usage: $0 {up|down|restart|logs|status}"
}

main() {
  check_deps
  write_env_if_missing

  local command="${1:-up}"
  case "${command}" in
    up)
      start_stack
      ;;
    down)
      stop_stack
      ;;
    restart)
      stop_stack
      start_stack
      ;;
    logs)
      show_logs
      ;;
    status)
      status
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
