#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

info() { printf "[INFO] %s\n" "$1"; }
error() { printf "[ERROR] %s\n" "$1" >&2; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    error "Required command not found: $1"
    exit 1
  fi
}

wait_for_postgres() {
  local container_name="$1"
  local max_wait_seconds=60
  local waited=0

  info "Waiting for PostgreSQL container to become healthy..."
  while true; do
    local status
    status="$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || true)"

    if [[ "$status" == "healthy" ]]; then
      info "PostgreSQL is healthy."
      break
    fi

    if (( waited >= max_wait_seconds )); then
      error "PostgreSQL did not become healthy within ${max_wait_seconds}s."
      docker compose logs postgres --tail=100 || true
      exit 1
    fi

    sleep 2
    waited=$((waited + 2))
  done
}

main() {
  require_cmd go
  require_cmd docker

  if ! docker compose version >/dev/null 2>&1; then
    error "Docker Compose plugin is required (docker compose)."
    exit 1
  fi

  info "Starting PostgreSQL with Docker Compose..."
  docker compose up -d

  wait_for_postgres "airbnb-scraper-postgres"

  info "Running Go scraper..."
  go run main.go

  info "Run complete."
}

main "$@"
