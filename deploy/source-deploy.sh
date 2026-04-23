#!/usr/bin/env bash

set -euo pipefail

# Build from a clean git snapshot in a temporary directory, then install only
# the final binary. This keeps the server checkout free of build artifacts.

REPO_DIR=""
INSTALL_DIR="${INSTALL_DIR:-/opt/sub2api}"
FRONTEND_DIR="${FRONTEND_DIR:-${INSTALL_DIR}/frontend}"
SERVICE_NAME="${SERVICE_NAME:-sub2api}"
BACKUP_BASE="${BACKUP_BASE:-${INSTALL_DIR}/upgrade-backups}"
VERIFY_URL="${VERIFY_URL:-http://127.0.0.1:8080/health}"
VERIFY_ATTEMPTS="${VERIFY_ATTEMPTS:-20}"
VERIFY_INTERVAL_SECONDS="${VERIFY_INTERVAL_SECONDS:-2}"
RESTART_SERVICE=1

export PATH="/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:${PATH:-}"

usage() {
  cat <<'EOF'
Usage:
  source-deploy.sh [options]

Options:
  --repo <path>         Source checkout path. Defaults to this script's repo.
  --install-dir <path>  Install directory. Default: /opt/sub2api
  --frontend-dir <path> Frontend static directory for Nginx. Default: /opt/sub2api/frontend
  --service <name>      systemd service name. Default: sub2api
  --no-restart          Install the binary but do not restart the service.
  -h, --help            Show this help.

Environment:
  INSTALL_DIR           Same as --install-dir.
  FRONTEND_DIR          Same as --frontend-dir.
  SERVICE_NAME          Same as --service.
  PNPM_BIN              pnpm executable to use. Defaults to corepack pnpm or pnpm.
  BACKUP_BASE           Backup directory. Defaults to /opt/sub2api/upgrade-backups.
  VERIFY_URL            Health check URL. Default: http://127.0.0.1:8080/health.

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
    --frontend-dir)
      FRONTEND_DIR="${2:-}"
      [[ -n "$FRONTEND_DIR" ]] || die "--frontend-dir requires a path"
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
FULL_COMMIT="$(git -C "$REPO_DIR" rev-parse HEAD)"
BRANCH="$(git -C "$REPO_DIR" branch --show-current 2>/dev/null || true)"
if [[ -z "$BRANCH" ]]; then
  BRANCH="detached"
fi
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
BACKUP_DIR="${BACKUP_BASE}/source_deploy_$(date +%Y%m%d_%H%M%S)"

echo "[1/6] Creating temporary build snapshot from $COMMIT..."
git -C "$REPO_DIR" archive --format=tar HEAD | tar -xf - -C "$BUILD_DIR"

VERSION="$(tr -d '\r\n' < "$BUILD_DIR/backend/cmd/server/VERSION")"

echo "[2/6] Building frontend in temporary directory..."
(
  cd "$BUILD_DIR/frontend"
  run_pnpm install --frozen-lockfile
  run_pnpm run build
)
FRONTEND_BUILD_DIR="$BUILD_DIR/backend/internal/web/dist"
[[ -f "$FRONTEND_BUILD_DIR/index.html" ]] || die "frontend build output missing index.html: $FRONTEND_BUILD_DIR"

echo "[3/6] Building embedded backend binary..."
(
  cd "$BUILD_DIR/backend"
  CGO_ENABLED=0 go build \
    -tags embed \
    -trimpath \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.Date=${DATE} -X main.BuildType=source" \
    -o "$BUILD_DIR/sub2api" \
    ./cmd/server
)

echo "[4/6] Installing frontend static files to $FRONTEND_DIR..."
mkdir -p "$FRONTEND_DIR"
cp -a "$FRONTEND_BUILD_DIR"/. "$FRONTEND_DIR"/
if id sub2api >/dev/null 2>&1; then
  chown -R sub2api:sub2api "$FRONTEND_DIR"
fi

echo "[5/6] Installing binary to $INSTALL_DIR/sub2api..."
mkdir -p "$INSTALL_DIR"
if [[ -f "$INSTALL_DIR/sub2api" ]]; then
  mkdir -p "$BACKUP_DIR"
  cp -a "$INSTALL_DIR/sub2api" "$BACKUP_DIR/sub2api"
  echo "Backed up previous binary to $BACKUP_DIR/sub2api"
fi
INSTALL_TMP="$INSTALL_DIR/.sub2api.new.$$"
install -m 0755 "$BUILD_DIR/sub2api" "$INSTALL_TMP"
if id sub2api >/dev/null 2>&1; then
  chown sub2api:sub2api "$INSTALL_TMP"
fi
mv -f "$INSTALL_TMP" "$INSTALL_DIR/sub2api"
printf '%s\n' "$BRANCH" > "$INSTALL_DIR/DEPLOYED_SOURCE_BRANCH"
printf '%s\n' "$FULL_COMMIT" > "$INSTALL_DIR/DEPLOYED_SOURCE_REF"

if [[ "$RESTART_SERVICE" -eq 1 ]]; then
  if command -v systemctl >/dev/null 2>&1; then
    echo "[6/6] Restarting $SERVICE_NAME..."
    systemctl restart "$SERVICE_NAME"
    echo "Verifying health at $VERIFY_URL..."
    HEALTH_OK=0
    for attempt in $(seq 1 "$VERIFY_ATTEMPTS"); do
      if curl -fsS "$VERIFY_URL" >/dev/null; then
        HEALTH_OK=1
        echo "Health check passed on attempt $attempt/$VERIFY_ATTEMPTS."
        break
      fi

      if [[ "$attempt" -lt "$VERIFY_ATTEMPTS" ]]; then
        sleep "$VERIFY_INTERVAL_SECONDS"
      fi
    done

    if [[ "$HEALTH_OK" -ne 1 ]]; then
      systemctl --no-pager --full status "$SERVICE_NAME" || true
      die "health check failed after $VERIFY_ATTEMPTS attempts: $VERIFY_URL"
    fi
  else
    echo "[6/6] systemctl not found; restart $SERVICE_NAME manually."
  fi
else
  echo "[6/6] Skipping service restart."
fi

if [[ -f "$REPO_DIR/sub2api" ]] && ! git -C "$REPO_DIR" ls-files --error-unmatch -- sub2api >/dev/null 2>&1; then
  echo "Note: existing checkout artifact remains at $REPO_DIR/sub2api; remove it once after verifying deployment."
fi

echo "Deployed Sub2API ${VERSION} (${COMMIT}) without writing build artifacts to $REPO_DIR."
