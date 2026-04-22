#!/usr/bin/env bash

set -euo pipefail

# Build from a clean git snapshot in a temporary directory, then install only
# the final binary. This keeps the server checkout free of build artifacts.

REPO_DIR=""
INSTALL_DIR="${INSTALL_DIR:-/opt/sub2api}"
SERVICE_NAME="${SERVICE_NAME:-sub2api}"
RESTART_SERVICE=1

usage() {
  cat <<'EOF'
Usage:
  source-deploy.sh [options]

Options:
  --repo <path>         Source checkout path. Defaults to this script's repo.
  --install-dir <path>  Install directory. Default: /opt/sub2api
  --service <name>      systemd service name. Default: sub2api
  --no-restart          Install the binary but do not restart the service.
  -h, --help            Show this help.

Environment:
  INSTALL_DIR           Same as --install-dir.
  SERVICE_NAME          Same as --service.
  PNPM_BIN              pnpm executable to use. Defaults to corepack pnpm or pnpm.

Example:
  sudo ./deploy/source-deploy.sh --repo /opt/sub2api-src
EOF
}

die() {
  echo "error: $*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO_DIR="${2:-}"
      [[ -n "$REPO_DIR" ]] || die "--repo requires a path"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:-}"
      [[ -n "$INSTALL_DIR" ]] || die "--install-dir requires a path"
      shift 2
      ;;
    --service)
      SERVICE_NAME="${2:-}"
      [[ -n "$SERVICE_NAME" ]] || die "--service requires a name"
      shift 2
      ;;
    --no-restart)
      RESTART_SERVICE=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -z "$REPO_DIR" ]]; then
  REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
else
  REPO_DIR="$(cd "$REPO_DIR" && pwd)"
fi

[[ -d "$REPO_DIR/.git" ]] || git -C "$REPO_DIR" rev-parse --git-dir >/dev/null 2>&1 || die "not a git checkout: $REPO_DIR"
[[ -f "$REPO_DIR/frontend/package.json" ]] || die "missing frontend/package.json under $REPO_DIR"
[[ -f "$REPO_DIR/backend/go.mod" ]] || die "missing backend/go.mod under $REPO_DIR"

if ! git -C "$REPO_DIR" diff --quiet -- . || ! git -C "$REPO_DIR" diff --cached --quiet -- .; then
  die "tracked source changes are not committed; commit or stash them before deploying"
fi

BUILD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-build.XXXXXX")"
cleanup() {
  rm -rf "$BUILD_DIR"
}
trap cleanup EXIT

run_pnpm() {
  if [[ -n "${PNPM_BIN:-}" ]]; then
    "$PNPM_BIN" "$@"
  elif command -v corepack >/dev/null 2>&1; then
    corepack pnpm "$@"
  else
    pnpm "$@"
  fi
}

COMMIT="$(git -C "$REPO_DIR" rev-parse --short=12 HEAD)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "[1/5] Creating temporary build snapshot from $COMMIT..."
git -C "$REPO_DIR" archive --format=tar HEAD | tar -xf - -C "$BUILD_DIR"

VERSION="$(tr -d '\r\n' < "$BUILD_DIR/backend/cmd/server/VERSION")"

echo "[2/5] Building frontend in temporary directory..."
(
  cd "$BUILD_DIR/frontend"
  run_pnpm install --frozen-lockfile
  run_pnpm run build
)

echo "[3/5] Building embedded backend binary..."
(
  cd "$BUILD_DIR/backend"
  CGO_ENABLED=0 go build \
    -tags embed \
    -trimpath \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.Date=${DATE} -X main.BuildType=source" \
    -o "$BUILD_DIR/sub2api" \
    ./cmd/server
)

echo "[4/5] Installing binary to $INSTALL_DIR/sub2api..."
mkdir -p "$INSTALL_DIR"
INSTALL_TMP="$INSTALL_DIR/.sub2api.new.$$"
install -m 0755 "$BUILD_DIR/sub2api" "$INSTALL_TMP"
if id sub2api >/dev/null 2>&1; then
  chown sub2api:sub2api "$INSTALL_TMP"
fi
mv -f "$INSTALL_TMP" "$INSTALL_DIR/sub2api"

if [[ "$RESTART_SERVICE" -eq 1 ]]; then
  if command -v systemctl >/dev/null 2>&1; then
    echo "[5/5] Restarting $SERVICE_NAME..."
    systemctl restart "$SERVICE_NAME"
    systemctl --no-pager --full status "$SERVICE_NAME" || true
  else
    echo "[5/5] systemctl not found; restart $SERVICE_NAME manually."
  fi
else
  echo "[5/5] Skipping service restart."
fi

if [[ -f "$REPO_DIR/sub2api" ]] && ! git -C "$REPO_DIR" ls-files --error-unmatch -- sub2api >/dev/null 2>&1; then
  echo "Note: existing checkout artifact remains at $REPO_DIR/sub2api; remove it once after verifying deployment."
fi

echo "Deployed Sub2API ${VERSION} (${COMMIT}) without writing build artifacts to $REPO_DIR."
