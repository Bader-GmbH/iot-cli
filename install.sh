#!/bin/bash
#
# Bader IoT CLI Installation Script
# Usage: curl -fsSL https://api.iot.bader.solutions/api/releases/cli/install.sh | bash
#

set -e

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="iot"
API_BASE_URL="${API_BASE_URL:-https://api.iot.bader.solutions}"
RELEASE_URL="${API_BASE_URL}/api/releases/cli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print functions
print_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}$1${NC}"
}

print_warning() {
    echo -e "${YELLOW}$1${NC}"
}

print_info() {
    echo "$1"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --install-dir=*)
                INSTALL_DIR="${1#*=}"
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
}

print_usage() {
    echo "Usage: curl -fsSL ${RELEASE_URL}/install.sh | bash [-s -- [options]]"
    echo ""
    echo "Optional arguments:"
    echo "  --install-dir=<path>  Installation directory (default: /usr/local/bin)"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  curl -fsSL ${RELEASE_URL}/install.sh | bash"
    echo "  curl -fsSL ${RELEASE_URL}/install.sh | bash -s -- --install-dir=\$HOME/.local/bin"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$ARCH" in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="arm"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    case "$OS" in
        linux|darwin)
            # Supported
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    print_info "Detected platform: $PLATFORM"
}

# Check write permissions and handle sudo if needed
check_permissions() {
    if [ ! -d "$INSTALL_DIR" ]; then
        print_info "Creating installation directory: $INSTALL_DIR"
        if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            if command -v sudo &> /dev/null; then
                sudo mkdir -p "$INSTALL_DIR"
            else
                print_error "Cannot create $INSTALL_DIR. Please run with sudo or specify a different directory."
                exit 1
            fi
        fi
    fi

    if [ ! -w "$INSTALL_DIR" ]; then
        if command -v sudo &> /dev/null; then
            USE_SUDO="sudo"
            print_info "Using sudo for installation to $INSTALL_DIR"
        else
            print_error "No write permission to $INSTALL_DIR. Please run with sudo or specify a different directory."
            exit 1
        fi
    else
        USE_SUDO=""
    fi
}

# Download the CLI binary
download_cli() {
    SUFFIX=""
    if [ "$OS" = "windows" ]; then
        SUFFIX=".exe"
    fi

    DOWNLOAD_URL="${RELEASE_URL}/latest/${BINARY_NAME}-${PLATFORM}${SUFFIX}"
    DOWNLOAD_PATH="${INSTALL_DIR}/${BINARY_NAME}${SUFFIX}"
    TMP_PATH=$(mktemp)

    print_info "Downloading CLI from $DOWNLOAD_URL..."

    # Download binary to temp location
    if command -v curl &> /dev/null; then
        HTTP_CODE=$(curl -fsSL -w "%{http_code}" "$DOWNLOAD_URL" -o "$TMP_PATH")
        if [ "$HTTP_CODE" != "200" ]; then
            print_error "Failed to download CLI (HTTP $HTTP_CODE)"
            rm -f "$TMP_PATH"
            exit 1
        fi
    elif command -v wget &> /dev/null; then
        wget -q "$DOWNLOAD_URL" -O "$TMP_PATH" || {
            print_error "Failed to download CLI"
            rm -f "$TMP_PATH"
            exit 1
        }
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    # Make executable
    chmod +x "$TMP_PATH"

    # Move to install directory
    $USE_SUDO mv "$TMP_PATH" "$DOWNLOAD_PATH"

    print_success "Downloaded CLI to $DOWNLOAD_PATH"
}

# Verify installation
verify_installation() {
    if [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        VERSION=$("${INSTALL_DIR}/${BINARY_NAME}" version 2>/dev/null | head -1 || echo "unknown")
        print_success "Installed: $VERSION"
    else
        print_error "Installation verification failed"
        exit 1
    fi
}

# Check if CLI is in PATH
check_path() {
    if ! command -v "$BINARY_NAME" &> /dev/null; then
        print_warning "Note: $INSTALL_DIR is not in your PATH"
        echo ""
        print_info "Add it to your PATH by running:"
        case "$(basename "$SHELL")" in
            zsh)
                print_info "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
                ;;
            bash)
                print_info "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc && source ~/.bashrc"
                ;;
            *)
                print_info "  export PATH=\"$INSTALL_DIR:\$PATH\""
                ;;
        esac
        echo ""
    fi
}

# Main installation flow
main() {
    echo ""
    echo "========================================"
    echo "  Bader IoT CLI Installer"
    echo "========================================"
    echo ""

    parse_args "$@"
    detect_platform
    check_permissions
    download_cli
    verify_installation
    check_path

    echo ""
    print_success "Installation complete!"
    echo ""
    print_info "Get started:"
    print_info "  ${BINARY_NAME} auth login    Authenticate with your account"
    print_info "  ${BINARY_NAME} device list   List your devices"
    print_info "  ${BINARY_NAME} --help        Show all commands"
    echo ""
}

# Run main with all arguments
main "$@"
