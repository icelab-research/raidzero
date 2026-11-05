#!/usr/bin/env bash

COMPOSE_FILE="$ROOT/compose.prod.yml"
ENV_DIR="$ROOT/env"
DATA_DIR="$ROOT/data"
STATE_DIR="$ROOT/state"
BIN_DIR="$ROOT/bin"
DEFAULT_LOG_LINES=200

OLD_DIRS=(
  /srv/frontend
  /srv/sf-voice-api
  /srv/resto-demo
  /srv/ellie-ai
  /srv/mysql
  /srv/qdrant
  /srv/redis
  /srv/caddy
)

usage() {
  cat <<'EOF'
usage:
  sfctl bootstrap
  sfctl inventory
  sfctl migrate-layout --dry-run
  sfctl migrate-layout --apply
  sfctl deploy <frontend|api|ellie|resto|caddy|mysql|redis|all> <tag>
  sfctl rollback <frontend|api|ellie|resto> <tag>
  sfctl restart <service|all>
  sfctl status [service|all]
  sfctl logs <service> [lines]
  sfctl smoke [service|all]
  sfctl preview deploy <preview-id> <api-tag> <frontend-tag> <host>
  sfctl preview destroy <preview-id>
  sfctl preview destroy-pr <pr-number>
  sfctl preview status <preview-id>
  sfctl preview logs <preview-id> [lines]
  sfctl preview smoke <preview-id>
  sfctl cleanup --dry-run
  sfctl cleanup --archive [YYYYMMDD]
  sfctl cleanup --delete-archive YYYYMMDD
EOF
}

die() {
  echo "sfctl: $*" >&2
  exit 1
}

need_root() {
  [[ "${EUID:-$(id -u)}" -eq 0 ]] || die "this command must run as root"
}

compose() {
  ensure_images_env
  docker compose --env-file "$ENV_DIR/images.env" -f "$COMPOSE_FILE" "$@"
}

service_name() {
  case "${1:-}" in
    ellie) echo "ellie-ai" ;;
    resto) echo "resto-demo" ;;
    all) echo "all" ;;
    *) die "unknown service: ${1:-<empty>} (core services moved to sf-voice/core)" ;;
  esac
}

image_var() {
  case "$1" in
    ellie) echo "ELLIE_IMAGE_TAG" ;;
    resto) echo "RESTO_IMAGE_TAG" ;;
    *) die "$1 does not have an image tag" ;;
  esac
}

image_ref() {
  case "$1" in
    ellie) echo "ghcr.io/icelab-research/raidzero/ellie-ai" ;;
    resto) echo "ghcr.io/icelab-research/raidzero/restaurant-booking-app" ;;
    *) die "$1 does not have an image" ;;
  esac
}

containers_for() {
  case "${1:-all}" in
    ellie) echo ellie-ai ;;
    resto) echo resto-demo ;;
    all) echo ellie-ai resto-demo ;;
    *) die "unknown service: $1" ;;
  esac
}

ensure_dirs() {
  mkdir -p \
    "$BIN_DIR" \
    "$ENV_DIR" \
    "$STATE_DIR/inventory" \
    "$DATA_DIR/resto" \
    "$DATA_DIR/ellie"
}

ensure_prod_networks() {
  docker network create proxy_net >/dev/null 2>&1 || true
}

ensure_images_env() {
  mkdir -p "$ENV_DIR"
  if [[ ! -f "$ENV_DIR/images.env" ]]; then
    cat > "$ENV_DIR/images.env" <<'EOF'
ELLIE_IMAGE_TAG=latest
RESTO_IMAGE_TAG=latest
EOF
  fi
}

set_image_tag() {
  local service="$1"
  local tag="$2"
  local var tmp
  var="$(image_var "$service")"
  tmp="$(mktemp)"
  ensure_images_env
  # flock serializes read-modify-write on images.env so concurrent deploys
  # of different services don't clobber each other's image tag. -e=exclusive,
  # -w 30 waits up to 30s for the lock — long enough to ride out a normal
  # deploy, short enough that a stuck lock surfaces as a clear error.
  exec 9>"$ENV_DIR/images.env.lock"
  flock -e -w 30 9 || die "set_image_tag: could not acquire images.env lock within 30s"
  awk -F= -v key="$var" -v value="$tag" '
    BEGIN { found = 0 }
    $1 == key { print key "=" value; found = 1; next }
    { print }
    END { if (!found) print key "=" value }
  ' "$ENV_DIR/images.env" > "$tmp"
  install -m 600 "$tmp" "$ENV_DIR/images.env"
  rm -f "$tmp"
  exec 9>&-
}

write_env_file() {
  local path="$1"
  shift
  local tmp
  tmp="$(mktemp)"
  umask 077
  for name in "$@"; do
    if [[ -n "${!name:-}" ]]; then
      printf '%s=%s\n' "$name" "${!name}" >> "$tmp"
    fi
  done
  install -m 600 "$tmp" "$path"
  rm -f "$tmp"
}

read_env_value() {
  # read KEY=value from a simple env file without sourcing it, so values
  # with shell metacharacters stay literal. prints the first match's value.
  local file="$1" key="$2" line
  [[ -f "$file" ]] || return 0
  line="$(grep -m1 "^${key}=" "$file" 2>/dev/null)" || return 0
  printf '%s' "${line#*=}"
}

default_data_urls() {
  # api creds must match what bootstrap actually generated on the droplet.
  # prefer an explicit secret; otherwise derive from the generated data
  # service env files so a missing/typo'd github secret can't desync auth.
  if [[ -z "${REDIS_URL:-}" ]]; then
    REDIS_URL="$(read_env_value "$ENV_DIR/redis.env" REDIS_URL)"
    [[ -n "$REDIS_URL" ]] && export REDIS_URL
  fi
  if [[ -z "${DATABASE_URL:-}" ]]; then
    local user pass db
    user="$(read_env_value "$ENV_DIR/mysql.env" MYSQL_USER)"
    pass="$(read_env_value "$ENV_DIR/mysql.env" MYSQL_PASSWORD)"
    db="$(read_env_value "$ENV_DIR/mysql.env" MYSQL_DATABASE)"
    if [[ -n "$user" && -n "$pass" && -n "$db" ]]; then
      DATABASE_URL="mysql://$user:$pass@mysql:3306/$db"
      export DATABASE_URL
    fi
  fi
}

write_service_env() {
  local service="$1"
  mkdir -p "$ENV_DIR"
  case "$service" in
    ellie)
      write_env_file "$ENV_DIR/ellie.env" \
        SECRET_KEY_BASE INTERNAL_API_TOKEN OPENAI_API_KEY TELNYX_API_KEY \
        TELNYX_PUBLIC_KEY PHONE_NUMBER STAFF_PHONE_E164 AWS_ACCESS_KEY_ID \
        AWS_SECRET_ACCESS_KEY AWS_REGION S3_BUCKET_NAME
      ;;
    resto)
      write_env_file "$ENV_DIR/resto.env" SECRET_KEY_BASE INTERNAL_API_TOKEN
      ;;
  esac
}
