#!/bin/bash
set -e

AIMA_BIN="/usr/local/bin/aima"
AIMA_CONFIG_DIR="/etc/aima"
AIMA_CONFIG="${AIMA_CONFIG_DIR}/aima.toml"
AIMA_DATA_DIR="/var/lib/aima"
AIMA_LOG_DIR="/var/log/aima"
AIMA_USER="aima"
AIMA_GROUP="aima"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

create_user() {
    if id "${AIMA_USER}" &>/dev/null; then
        log_info "User ${AIMA_USER} already exists"
    else
        useradd --system --no-create-home --shell /bin/false "${AIMA_USER}"
        log_info "Created user ${AIMA_USER}"
    fi
}

create_directories() {
    mkdir -p "${AIMA_CONFIG_DIR}"
    mkdir -p "${AIMA_DATA_DIR}"
    mkdir -p "${AIMA_LOG_DIR}"
    
    chown "${AIMA_USER}:${AIMA_GROUP}" "${AIMA_DATA_DIR}"
    chown "${AIMA_USER}:${AIMA_GROUP}" "${AIMA_LOG_DIR}"
    
    log_info "Created directories"
}

install_binary() {
    if [ -f "./aima" ]; then
        cp ./aima "${AIMA_BIN}"
        chmod +x "${AIMA_BIN}"
        log_info "Installed binary to ${AIMA_BIN}"
    elif [ -f "./bin/aima" ]; then
        cp ./bin/aima "${AIMA_BIN}"
        chmod +x "${AIMA_BIN}"
        log_info "Installed binary to ${AIMA_BIN}"
    else
        log_warn "Binary not found in ./aima or ./bin/aima"
        log_warn "Please ensure the binary is built before running this script"
    fi
}

create_default_config() {
    if [ ! -f "${AIMA_CONFIG}" ]; then
        cat > "${AIMA_CONFIG}" << EOF
# AIMA Configuration File
# See docs for configuration options

[server]
host = "0.0.0.0"
port = 8080

[log]
level = "info"
EOF
        log_info "Created default config at ${AIMA_CONFIG}"
    else
        log_info "Config file already exists at ${AIMA_CONFIG}"
    fi
}

install_systemd_service() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SERVICE_FILE="${SCRIPT_DIR}/aima.service"
    
    if [ ! -f "${SERVICE_FILE}" ]; then
        log_error "Service file not found: ${SERVICE_FILE}"
        exit 1
    fi
    
    cp "${SERVICE_FILE}" /etc/systemd/system/aima.service
    systemctl daemon-reload
    log_info "Installed systemd service"
}

enable_service() {
    systemctl enable aima.service
    log_info "Enabled aima.service to start on boot"
}

print_usage() {
    echo ""
    echo "Installation complete!"
    echo ""
    echo "Usage:"
    echo "  sudo systemctl start aima    # Start the service"
    echo "  sudo systemctl stop aima     # Stop the service"
    echo "  sudo systemctl status aima   # Check service status"
    echo "  sudo journalctl -u aima      # View logs"
    echo ""
}

main() {
    echo "======================================"
    echo "  AIMA Service Installer"
    echo "======================================"
    echo ""
    
    check_root
    create_user
    create_directories
    install_binary
    create_default_config
    install_systemd_service
    enable_service
    print_usage
    
    log_info "Installation completed successfully!"
}

main "$@"
