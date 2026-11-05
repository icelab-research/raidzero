#!/usr/bin/env bash
# thin entrypoint for the production droplet control surface.

set -euo pipefail

ROOT="${SF_ROOT:-/srv/sf-voice}"
RAW_BASE="${RAW_BASE:-https://raw.githubusercontent.com/icelab-research/raidzero/main/infra/deploy}"

load_libs() {
  local repo_lib_dir
  repo_lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd)/sfctl.d"

  if [[ -d "$repo_lib_dir" ]]; then
    # shellcheck source=/dev/null
    for file in "$repo_lib_dir"/*.sh; do . "$file"; done
    return
  fi

  local installed_lib_dir="$ROOT/bin/sfctl.d"
  if [[ -d "$installed_lib_dir" ]]; then
    # shellcheck source=/dev/null
    for file in "$installed_lib_dir"/*.sh; do . "$file"; done
    return
  fi

  local tmp
  tmp="$(mktemp -d)"
  for file in common bootstrap migrate deploy preview; do
    curl -fsSL "$RAW_BASE/sfctl.d/$file.sh" -o "$tmp/$file.sh"
  done
  # shellcheck source=/dev/null
  for file in "$tmp"/*.sh; do . "$file"; done
}

main() {
  local cmd="${1:-}"
  shift || true
  case "$cmd" in
    bootstrap) bootstrap ;;
    inventory) inventory ;;
    migrate-layout) migrate_layout "${1:---dry-run}" ;;
    deploy) deploy_service "${1:-}" "${2:-}" ;;
    rollback) deploy_service "${1:-}" "${2:-}" ;;
    restart) restart_service "${1:-all}" ;;
    status) status_service "${1:-all}" ;;
    logs) logs_service "${1:-all}" "${2:-$DEFAULT_LOG_LINES}" ;;
    smoke) smoke "${1:-all}" ;;
    preview) preview "$@" ;;
    cleanup) cleanup "${1:---dry-run}" "${2:-}" ;;
    mysql-backup) mysql_backup ;;
    help|-h|--help|"") usage ;;
    *) usage; die "unknown command: $cmd" ;;
  esac
}

load_libs
main "$@"
