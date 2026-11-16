#!/usr/bin/env bash
# Build the API image on the Mac for the Pi's architecture, ship it over SSH,
# and restart the container.
#
# Usage:
#   Copy deploy.env.example to deploy.env and set PI_SSH there,
#   or pass variables in the environment:
#   PI_SSH=<user>@<pi-ip> ./deploy.sh
#
# Assumes:
#   - The Pi is 64-bit (aarch64) and runs Docker with the compose plugin
#   - The current user can SSH into the Pi without a password prompt
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# deploy.env is gitignored; it holds the Pi-specific defaults
[ -e "$SCRIPT_DIR/deploy.env" ] && source "$SCRIPT_DIR/deploy.env"

PI_SSH="${PI_SSH:?PI_SSH not set — put it in server/deploy/deploy.env or the environment}"
PI_DIR="${PI_DIR:-pi-alarm}"   # home-relative directory on the Pi
PLATFORM="${PLATFORM:-linux/arm64}"
IMAGE="pi-alarm-api:latest"

TAR_FILE="$(mktemp -t pi-alarm-api)"

# Reuse a single SSH connection for all steps, so a password
# prompt (before key auth is set up) happens at most once per deploy.
# %C is a short hash of the connection parameters, keeping the socket
# path within macOS's ~104-char Unix domain socket limit.
SSH_CONTROL="/tmp/ssh-pi-alarm-%C"
SSH_OPTS=(-o ControlMaster=auto -o ControlPath="$SSH_CONTROL" -o ControlPersist=1m)

cleanup() { rm -f "$TAR_FILE"; ssh "${SSH_OPTS[@]}" -O exit "$PI_SSH" 2>/dev/null || true; }
trap cleanup EXIT

echo "==> Building $IMAGE for $PLATFORM"
docker buildx build \
    --platform "$PLATFORM" \
    -t "$IMAGE" \
    --load \
    "$SERVER_DIR"

echo "==> Saving image"
docker save "$IMAGE" -o "$TAR_FILE"

echo "==> Copying image and config to $PI_SSH"
ssh "${SSH_OPTS[@]}" "$PI_SSH" "mkdir -p ~/$PI_DIR/data"
scp -o ControlMaster=auto -o ControlPath="$SSH_CONTROL" -o ControlPersist=1m \
    "$TAR_FILE" "$PI_SSH:~/$PI_DIR/pi-alarm-api.tar"
scp -o ControlMaster=auto -o ControlPath="$SSH_CONTROL" -o ControlPersist=1m \
    "$SCRIPT_DIR/docker-compose.yml" "$PI_SSH:~/$PI_DIR/docker-compose.yml"
[ -e "$SERVER_DIR/.env" ] && \
    scp -o ControlMaster=auto -o ControlPath="$SSH_CONTROL" -o ControlPersist=1m \
    "$SERVER_DIR/.env" "$PI_SSH:~/$PI_DIR/.env"

echo "==> Loading image and restarting container"
ssh "${SSH_OPTS[@]}" "$PI_SSH" "cd ~/$PI_DIR \
    && docker load -i pi-alarm-api.tar \
    && rm pi-alarm-api.tar \
    && docker compose up -d --force-recreate"

echo "==> Deployed. Recent logs:"
ssh "${SSH_OPTS[@]}" "$PI_SSH" "cd ~/$PI_DIR && docker compose logs --tail 20 api"
