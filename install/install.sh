#!/usr/bin/env bash
# Easy Plan installer.
#
#   curl -fsSL https://raw.githubusercontent.com/hrrydgls/easy-plan/main/install/install.sh | bash
#   curl -fsSL .../install.sh | bash -s -- --port 9092 --close-registration
#
# Downloads the easyplan binary, creates an unprivileged system user, and
# installs+starts a systemd service. Re-running it upgrades the binary and
# keeps all data. Linux + systemd only.

set -euo pipefail

REPO="${EASYPLAN_REPO:-hrrydgls/easy-plan}"
VERSION="${EASYPLAN_VERSION:-latest}"
PORT="${EASYPLAN_PORT:-9092}"
REGISTRATION="${EASYPLAN_REGISTRATION:-true}"
LOCAL_BINARY=""
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/easyplan"
CONFIG_DIR="/etc/easyplan"
ENV_FILE="$CONFIG_DIR/easyplan.env"
SERVICE_FILE="/etc/systemd/system/easyplan.service"
SERVICE_USER="easyplan"
BIN_NAME="easyplan"

log()  { printf '\033[1;32m==>\033[0m %s\n' "$1"; }
warn() { printf '\033[1;33m==>\033[0m %s\n' "$1" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

usage() {
	cat <<EOF
Usage: install.sh [options]

  --port <n>              Port to listen on (default: 9092)
  --version <tag>         Release to install, e.g. v0.1.0 (default: latest)
  --close-registration    Don't let new people sign up (default: open)
  --open-registration     Let new people sign up
  --binary <path>         Install a binary you built yourself instead of downloading
  -h, --help              Show this help
EOF
}

while [ $# -gt 0 ]; do
	case "$1" in
	--port) PORT="$2"; shift 2 ;;
	--version) VERSION="$2"; shift 2 ;;
	--close-registration) REGISTRATION="false"; shift ;;
	--open-registration) REGISTRATION="true"; shift ;;
	--binary) LOCAL_BINARY="$2"; shift 2 ;;
	-h | --help) usage; exit 0 ;;
	*) die "unknown argument: $1 (see --help)" ;;
	esac
done

[ "$(uname -s)" = "Linux" ] || die "Easy Plan only supports Linux"
command -v systemctl >/dev/null 2>&1 || die "systemd (systemctl) is required"
case "$PORT" in '' | *[!0-9]*) die "--port must be a number" ;; esac

case "$(uname -m)" in
x86_64 | amd64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
*) die "unsupported architecture: $(uname -m)" ;;
esac

if [ "$(id -u)" -eq 0 ]; then
	SUDO=""
else
	command -v sudo >/dev/null 2>&1 || die "run as root or install sudo"
	SUDO="sudo"
fi

UPGRADE=""
[ -f "$SERVICE_FILE" ] && UPGRADE=1

TMP_BIN="$(mktemp)"
trap 'rm -f "$TMP_BIN"' EXIT
if [ -n "$LOCAL_BINARY" ]; then
	[ -f "$LOCAL_BINARY" ] || die "binary not found: $LOCAL_BINARY"
	log "Using local binary $LOCAL_BINARY"
	cp "$LOCAL_BINARY" "$TMP_BIN"
else
	command -v curl >/dev/null 2>&1 || die "curl is required"
	if [ "$VERSION" = "latest" ]; then
		URL="https://github.com/${REPO}/releases/latest/download/easyplan-linux-${ARCH}"
	else
		URL="https://github.com/${REPO}/releases/download/${VERSION}/easyplan-linux-${ARCH}"
	fi
	log "Downloading easyplan ($ARCH, $VERSION)..."
	curl -fsSL -o "$TMP_BIN" "$URL" || die "download failed: $URL (has a release been published yet?)"
fi
chmod 755 "$TMP_BIN"

if ! id "$SERVICE_USER" >/dev/null 2>&1; then
	log "Creating system user '$SERVICE_USER'..."
	$SUDO useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
fi

$SUDO mkdir -p "$DATA_DIR" "$CONFIG_DIR"
$SUDO chown "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"
$SUDO chmod 700 "$DATA_DIR"

if [ -n "$UPGRADE" ]; then
	log "Existing install found, upgrading (your data in $DATA_DIR is kept)..."
	$SUDO systemctl stop easyplan || true
fi
$SUDO install -m 755 "$TMP_BIN" "$INSTALL_DIR/$BIN_NAME"
log "Installed binary to $INSTALL_DIR/$BIN_NAME"

$SUDO tee "$ENV_FILE" >/dev/null <<EOF
EASYPLAN_PORT=$PORT
EASYPLAN_DATA_DIR=$DATA_DIR
EASYPLAN_REGISTRATION=$REGISTRATION
# Set to true when Easy Plan is served over HTTPS (e.g. behind Caddy/nginx).
EASYPLAN_SECURE_COOKIES=false
EOF
$SUDO chown root:"$SERVICE_USER" "$ENV_FILE"
$SUDO chmod 640 "$ENV_FILE"

log "Writing systemd unit..."
$SUDO tee "$SERVICE_FILE" >/dev/null <<EOF
[Unit]
Description=Easy Plan calendar
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
EnvironmentFile=$ENV_FILE
ExecStart=$INSTALL_DIR/$BIN_NAME
Restart=on-failure
RestartSec=2
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

$SUDO systemctl daemon-reload
$SUDO systemctl enable easyplan >/dev/null 2>&1
$SUDO systemctl restart easyplan

log "Waiting for Easy Plan to start..."
ok=""
for _ in $(seq 1 15); do
	if curl -fsS -o /dev/null "http://127.0.0.1:$PORT/api/config" 2>/dev/null; then
		ok=1
		break
	fi
	sleep 1
done
if [ -z "$ok" ]; then
	warn "Easy Plan didn't come up in time. Check what's wrong with:"
	warn "  systemctl status easyplan"
	warn "  journalctl -u easyplan -e"
	exit 1
fi

IP="$(curl -fsS --max-time 3 https://ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')"

echo
log "Easy Plan is running."
echo "  URL:           http://${IP:-<this-server-ip>}:$PORT"
echo "  Registration:  $([ "$REGISTRATION" = "true" ] && echo "open — create your account at /register" || echo "closed")"
echo
echo "It's a systemd service, so it starts on every boot and restarts if it"
echo "crashes. Re-run this installer any time to upgrade."
echo
echo "Settings:  $ENV_FILE  (then: systemctl restart easyplan)"
echo "Data:      $DATA_DIR/easyplan.db"
echo "Manage:    systemctl {status|restart|stop} easyplan"
echo "Logs:      journalctl -u easyplan -f"
