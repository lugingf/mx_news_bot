#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="${APP_NAME:-mx_news_bot}"
APP_DIR="${APP_DIR:-/opt/mx_news_bot}"
NETWORK="${NETWORK:-${APP_NAME}_net}"
# The bot reads everything it shows from lap_vision, which runs on this host behind its own
# docker network. host.docker.internal is no help: it resolves to the bridge gateway, while the
# lap_vision proxy is published on the loopback interface only. So the bot joins that network as
# a second one and reaches the proxy by container name, the same way the database is shared.
BACKEND_NETWORK="${BACKEND_NETWORK:-lapvision_net}"
PUBLIC_PORT="${PUBLIC_PORT:-8585}"
PUBLIC_BIND_ADDR="${PUBLIC_BIND_ADDR:-127.0.0.1}"
IMAGE="${IMAGE:?IMAGE is required}"
HEALTH_PATH="${HEALTH_PATH:-/healthz}"
HEALTH_ATTEMPTS="${HEALTH_ATTEMPTS:-30}"
HEALTH_SLEEP_SECONDS="${HEALTH_SLEEP_SECONDS:-2}"
BOT_PORT="${BOT_PORT:-8085}"
METRICS_PORT="${METRICS_PORT:-9595}"
METRICS_PATH="${METRICS_PATH:-/metrics-mx}"
WEBHOOK_PATH="${WEBHOOK_PATH:?WEBHOOK_PATH is required}"
ENABLE_HOST_GATEWAY="${ENABLE_HOST_GATEWAY:-true}"

GHCR_USERNAME="${GHCR_USERNAME:-}"
GHCR_TOKEN="${GHCR_TOKEN:-}"

BLUE_NAME="${APP_NAME}-blue"
GREEN_NAME="${APP_NAME}-green"
PROXY_NAME="${APP_NAME}-proxy"
ACTIVE_FILE="${APP_DIR}/active_color"
CONFIG_PATH="${APP_DIR}/config.json"
NGINX_DIR="${APP_DIR}/nginx"
NGINX_CONF="${NGINX_DIR}/default.conf"

log() {
  printf '[deploy] %s\n' "$*"
}

docker_run_extra_args=()
if [[ "${ENABLE_HOST_GATEWAY}" == "true" ]]; then
  docker_run_extra_args+=(--add-host "host.docker.internal:host-gateway")
fi

mkdir -p "${APP_DIR}" "${NGINX_DIR}"

if [[ ! -f "${CONFIG_PATH}" ]]; then
  log "missing config file: ${CONFIG_PATH}"
  exit 1
fi
chmod 644 "${CONFIG_PATH}" || true

# These become nginx `location =` values, so they have to be request paths and nothing else.
# Prefixing a slash onto whatever arrived was wrong: given a full URL it produced
# `location = /https://host/path`, which nginx accepts happily and which no request ever matches,
# so Telegram got a 404 that looked like a routing bug rather than a bad variable.
require_request_path() {
  local name="$1" value="$2"

  if [[ -z "${value}" ]]; then
    log "${name} is empty"
    exit 1
  fi
  if [[ "${value}" == *"://"* || "${value}" == *" "* ]]; then
    log "${name} must be a request path such as /mxbotwebhook, got: ${value}"
    exit 1
  fi
  if [[ "${value:0:1}" != "/" ]]; then
    log "${name} must start with a slash, got: ${value}"
    exit 1
  fi
}

if [[ -z "${METRICS_PATH}" ]]; then
  METRICS_PATH="/metrics-mx"
fi

require_request_path WEBHOOK_PATH "${WEBHOOK_PATH}"
require_request_path METRICS_PATH "${METRICS_PATH}"

if [[ -n "${GHCR_USERNAME}" && -n "${GHCR_TOKEN}" ]]; then
  log "login to ghcr.io"
  echo "${GHCR_TOKEN}" | docker login ghcr.io -u "${GHCR_USERNAME}" --password-stdin
fi

if ! docker network inspect "${NETWORK}" >/dev/null 2>&1; then
  log "create docker network ${NETWORK}"
  docker network create "${NETWORK}"
fi

log "pull image ${IMAGE}"
docker pull "${IMAGE}"

# Applied once, before any candidate serves traffic, using the image being deployed: the
# migrations are baked into it, so they always match the code about to run.
log "run database migrations"
docker run --rm \
  --network "${NETWORK}" \
  "${docker_run_extra_args[@]}" \
  -v "${CONFIG_PATH}:/app/config.json:ro" \
  "${IMAGE}" \
  -config /app/config.json \
  -migrate-up \
  -migrations /app/migrations

active=""
if [[ -f "${ACTIVE_FILE}" ]]; then
  active="$(cat "${ACTIVE_FILE}" || true)"
fi
if [[ "${active}" != "blue" && "${active}" != "green" ]]; then
  if docker ps --format '{{.Names}}' | grep -qx "${BLUE_NAME}"; then
    active="blue"
  elif docker ps --format '{{.Names}}' | grep -qx "${GREEN_NAME}"; then
    active="green"
  else
    active="green"
  fi
fi

if [[ "${active}" == "blue" ]]; then
  next="green"
  old="blue"
else
  next="blue"
  old="green"
fi

new_name="${APP_NAME}-${next}"
old_name="${APP_NAME}-${old}"

log "active=${active} next=${next}"

if docker ps -a --format '{{.Names}}' | grep -qx "${new_name}"; then
  log "remove previous ${new_name}"
  docker rm -f "${new_name}" >/dev/null
fi

log "start candidate container ${new_name}"
docker run -d \
  --name "${new_name}" \
  --network "${NETWORK}" \
  "${docker_run_extra_args[@]}" \
  --restart unless-stopped \
  -v "${CONFIG_PATH}:/app/config.json:ro" \
  "${IMAGE}"

if [[ -n "${BACKEND_NETWORK}" ]] && docker network inspect "${BACKEND_NETWORK}" >/dev/null 2>&1; then
  log "attach ${new_name} to ${BACKEND_NETWORK}"
  docker network connect "${BACKEND_NETWORK}" "${new_name}" >/dev/null 2>&1 || true
else
  log "backend network ${BACKEND_NETWORK} not found; the bot will not reach lap_vision"
fi

healthy=0
for i in $(seq 1 "${HEALTH_ATTEMPTS}"); do
  if ! docker ps --format '{{.Names}}' | grep -qx "${new_name}"; then
    log "candidate ${new_name} is not running"
    break
  fi
  # Probed on the metrics port rather than the bot port: the bot port only answers Telegram
  # webhook posts, which a deploy cannot forge.
  if docker run --rm --network "${NETWORK}" curlimages/curl:8.12.1 \
    -fsS "http://${new_name}:${METRICS_PORT}${HEALTH_PATH}" >/dev/null; then
    healthy=1
    break
  fi
  sleep "${HEALTH_SLEEP_SECONDS}"
done

if [[ "${healthy}" != "1" ]]; then
  log "candidate ${new_name} is unhealthy"
  docker ps -a --filter "name=${new_name}" --format 'name={{.Names}} status={{.Status}}' || true
  docker logs "${new_name}" | tail -n 200 || true
  docker rm -f "${new_name}" >/dev/null || true
  exit 1
fi

log "write nginx upstream to ${new_name}"
cat > "${NGINX_CONF}" <<NGINX
server {
    listen 80;

    location = ${WEBHOOK_PATH} {
        proxy_pass http://${new_name}:${BOT_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$http_x_real_ip;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # lap_vision posts publication requests here. This proxy is bound to the loopback interface
    # and the host's nginx does not forward the path, so it is reachable only from this machine;
    # the request is additionally HMAC-signed.
    location = /internal/publications {
        proxy_pass http://${new_name}:${METRICS_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header Connection "";
    }

    location = ${METRICS_PATH} {
        proxy_pass http://${new_name}:${METRICS_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$http_x_real_ip;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location / {
        return 404;
    }
}
NGINX

if docker ps -a --format '{{.Names}}' | grep -qx "${PROXY_NAME}"; then
  docker rm -f "${PROXY_NAME}" >/dev/null
fi

log "start proxy container ${PROXY_NAME} on ${PUBLIC_BIND_ADDR}:${PUBLIC_PORT}"
docker run -d \
  --name "${PROXY_NAME}" \
  --network "${NETWORK}" \
  --restart unless-stopped \
  -p "${PUBLIC_BIND_ADDR}:${PUBLIC_PORT}:80" \
  -v "${NGINX_CONF}:/etc/nginx/conf.d/default.conf:ro" \
  nginx:1.27-alpine

echo "${next}" > "${ACTIVE_FILE}"

if docker ps --format '{{.Names}}' | grep -qx "${old_name}"; then
  log "stop old container ${old_name}"
  docker rm -f "${old_name}" >/dev/null || true
fi

log "deployment completed successfully"
docker image prune -af || true
docker builder prune -af || true
