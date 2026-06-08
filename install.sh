#!/bin/bash
# fpm Interactive Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | bash
# Non-interactive: curl ... | bash -s -- --yes
# Custom dir: FPM_INSTALL_DIR=/custom/path curl ... | bash

set -e

# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────

REPO="Kartikey2011yadav/fpm"
BINARY_NAME="fpm"
NONINTERACTIVE=false

# Colors (respect NO_COLOR)
if [ -z "$NO_COLOR" ] && [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    DIM='\033[2m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' CYAN='' BOLD='' DIM='' NC=''
fi

# ─────────────────────────────────────────────────────────────────────────────
# Parse Arguments
# ─────────────────────────────────────────────────────────────────────────────

while [ $# -gt 0 ]; do
    case "$1" in
        --yes|-y) NONINTERACTIVE=true ;;
        --help|-h)
            echo "fpm installer"
            echo "  --yes, -y    Non-interactive (use defaults)"
            echo "  --help, -h   Show this help"
            echo ""
            echo "Environment variables:"
            echo "  FPM_INSTALL_DIR          Binary location (default: ~/.local/bin)"
            echo "  FPM_CACHE_DIR            Cache directory"
            echo "  FPM_DATA_DIR             Data directory"
            echo "  FPM_INSTALL_NONINTERACTIVE=1  Same as --yes"
            exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
    shift
done

# Non-interactive if piped or env var set
if [ ! -t 0 ] || [ "${FPM_INSTALL_NONINTERACTIVE}" = "1" ]; then
    NONINTERACTIVE=true
fi

# ─────────────────────────────────────────────────────────────────────────────
# Platform Detection
# ─────────────────────────────────────────────────────────────────────────────

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported OS: $OS"; exit 1 ;;
    esac

    PLATFORM="${OS}-${ARCH}"
}

detect_shell() {
    SHELL_NAME=$(basename "${SHELL:-/bin/bash}")
    case "$SHELL_NAME" in
        zsh) SHELL_PROFILE="$HOME/.zshrc" ;;
        fish) SHELL_PROFILE="$HOME/.config/fish/config.fish" ;;
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                SHELL_PROFILE="$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                SHELL_PROFILE="$HOME/.bash_profile"
            else
                SHELL_PROFILE="$HOME/.profile"
            fi ;;
        *) SHELL_PROFILE="$HOME/.profile" ;;
    esac
}

# ─────────────────────────────────────────────────────────────────────────────
# UI Helpers
# ─────────────────────────────────────────────────────────────────────────────

banner() {
    echo ""
    echo -e "${BOLD}${CYAN}"
    echo '   ███████╗██████╗ ███╗   ███╗'
    echo '   ██╔════╝██╔══██╗████╗ ████║'
    echo '   █████╗  ██████╔╝██╔████╔██║'
    echo '   ██╔══╝  ██╔═══╝ ██║╚██╔╝██║'
    echo '   ██║     ██║     ██║ ╚═╝ ██║'
    echo '   ╚═╝     ╚═╝     ╚═╝     ╚═╝'
    echo -e "${NC}"
    echo -e "   ${BOLD}Fast Package Manager for Python${NC}"
    echo -e "   ${DIM}Platform: ${PLATFORM}${NC}"
    echo ""
}

info() { echo -e "  ${GREEN}✓${NC} $1"; }
warn() { echo -e "  ${YELLOW}●${NC} $1"; }
error() { echo -e "  ${RED}✗${NC} $1" >&2; }
step() { echo -e "  ${BLUE}↓${NC} $1"; }

get_disk_space() {
    local dir="$1"
    # Get available space in human-readable form
    while [ ! -d "$dir" ] && [ "$dir" != "/" ]; do
        dir=$(dirname "$dir")
    done
    df -h "$dir" 2>/dev/null | awk 'NR==2{print $4}' || echo "unknown"
}

# Simple menu: returns selected index (0-based)
# Usage: select_option "prompt" "option1" "option2" ...
select_option() {
    local prompt="$1"
    shift
    local options=("$@")
    local selected=0
    local count=${#options[@]}

    echo -e "\n${BOLD}? ${prompt}${NC}"

    if [ "$NONINTERACTIVE" = true ]; then
        echo -e "  ${GREEN}●${NC} ${options[0]} ${DIM}(auto-selected)${NC}"
        SELECTED_INDEX=0
        return
    fi

    # Draw options
    local i
    for i in $(seq 0 $((count - 1))); do
        if [ $i -eq $selected ]; then
            echo -e "  ${GREEN}●${NC} ${options[$i]}"
        else
            echo -e "    ${options[$i]}"
        fi
    done

    # Read input
    while true; do
        read -rsn1 key
        case "$key" in
            A|k) # Up arrow or k
                selected=$(( (selected - 1 + count) % count ))
                ;;
            B|j) # Down arrow or j
                selected=$(( (selected + 1) % count ))
                ;;
            '') # Enter
                break
                ;;
        esac

        # Redraw
        echo -en "\033[${count}A"
        for i in $(seq 0 $((count - 1))); do
            echo -en "\033[2K"
            if [ $i -eq $selected ]; then
                echo -e "  ${GREEN}●${NC} ${options[$i]}"
            else
                echo -e "    ${options[$i]}"
            fi
        done
    done

    SELECTED_INDEX=$selected
}

prompt_custom_path() {
    local prompt="$1"
    local default="$2"
    echo -en "  ${BOLD}${prompt}${NC} ${DIM}[${default}]${NC}: "
    if [ "$NONINTERACTIVE" = true ]; then
        echo "$default"
        CUSTOM_PATH="$default"
        return
    fi
    read -r input
    CUSTOM_PATH="${input:-$default}"
}

# ─────────────────────────────────────────────────────────────────────────────
# Default Paths
# ─────────────────────────────────────────────────────────────────────────────

set_defaults() {
    # Binary location
    if [ -n "$FPM_INSTALL_DIR" ]; then
        DEFAULT_BIN_DIR="$FPM_INSTALL_DIR"
    elif [ -w "/usr/local/bin" ]; then
        DEFAULT_BIN_DIR="/usr/local/bin"
    else
        DEFAULT_BIN_DIR="$HOME/.local/bin"
    fi

    # XDG-compliant defaults
    case "$OS" in
        darwin)
            DEFAULT_CACHE_DIR="${FPM_CACHE_DIR:-$HOME/Library/Caches/fpm}"
            DEFAULT_DATA_DIR="${FPM_DATA_DIR:-$HOME/Library/Application Support/fpm}"
            DEFAULT_CONFIG_DIR="${FPM_CONFIG_DIR:-$HOME/Library/Application Support/fpm}"
            ;;
        *)
            DEFAULT_CACHE_DIR="${FPM_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/fpm}"
            DEFAULT_DATA_DIR="${FPM_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/fpm}"
            DEFAULT_CONFIG_DIR="${FPM_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/fpm}"
            ;;
    esac

    DEFAULT_TOOL_BIN="$HOME/.local/bin"

    # Detect JupyterHub environment
    if [ -n "$JUPYTERHUB_USER" ]; then
        warn "JupyterHub detected (user: $JUPYTERHUB_USER)"
        DEFAULT_BIN_DIR="$HOME/.local/bin"
    fi

    # Use defaults
    BIN_DIR="$DEFAULT_BIN_DIR"
    CACHE_DIR="$DEFAULT_CACHE_DIR"
    DATA_DIR="$DEFAULT_DATA_DIR"
    CONFIG_DIR="$DEFAULT_CONFIG_DIR"
    TOOL_BIN_DIR="$DEFAULT_TOOL_BIN"
}

# ─────────────────────────────────────────────────────────────────────────────
# Installation Modes
# ─────────────────────────────────────────────────────────────────────────────

show_defaults() {
    echo ""
    echo -e "  ${BOLD}Installation paths:${NC}"
    echo -e "    Binary:     ${BIN_DIR}/fpm"
    echo -e "    Cache:      ${CACHE_DIR} ${DIM}(packages, wheels)${NC}"
    echo -e "    Data:       ${DATA_DIR} ${DIM}(Python versions, tools)${NC}"
    echo -e "    Config:     ${CONFIG_DIR} ${DIM}(settings)${NC}"
    echo -e "    Tool bin:   ${TOOL_BIN_DIR} ${DIM}(added to PATH)${NC}"
    echo ""
    local space=$(get_disk_space "$HOME")
    echo -e "  ${DIM}Disk: ~15 MB binary, cache grows with use (${space} available)${NC}"
}

run_custom_mode() {
    local cache_space=$(get_disk_space "$DEFAULT_CACHE_DIR")

    # Binary path
    select_option "Binary installation path" \
        "/usr/local/bin (system-wide, may need sudo)" \
        "$HOME/.local/bin (user only, no sudo)" \
        "Custom path..."
    case $SELECTED_INDEX in
        0) BIN_DIR="/usr/local/bin" ;;
        1) BIN_DIR="$HOME/.local/bin" ;;
        2) prompt_custom_path "Enter path" "$DEFAULT_BIN_DIR"; BIN_DIR="$CUSTOM_PATH" ;;
    esac

    # Cache dir
    select_option "Cache directory (packages, metadata)" \
        "$DEFAULT_CACHE_DIR (default, ${cache_space} available)" \
        "/tmp/fpm-cache (temporary)" \
        "Custom path..."
    case $SELECTED_INDEX in
        0) CACHE_DIR="$DEFAULT_CACHE_DIR" ;;
        1) CACHE_DIR="/tmp/fpm-cache" ;;
        2) prompt_custom_path "Enter path" "$DEFAULT_CACHE_DIR"; CACHE_DIR="$CUSTOM_PATH" ;;
    esac

    # Data dir
    select_option "Data directory (Python versions, tools)" \
        "$DEFAULT_DATA_DIR (default)" \
        "Custom path..."
    case $SELECTED_INDEX in
        0) DATA_DIR="$DEFAULT_DATA_DIR" ;;
        1) prompt_custom_path "Enter path" "$DEFAULT_DATA_DIR"; DATA_DIR="$CUSTOM_PATH" ;;
    esac

    # Tool bin
    select_option "Tool bin directory (tool executables)" \
        "$HOME/.local/bin (default, in PATH)" \
        "/usr/local/bin (system-wide)" \
        "Custom path..."
    case $SELECTED_INDEX in
        0) TOOL_BIN_DIR="$HOME/.local/bin" ;;
        1) TOOL_BIN_DIR="/usr/local/bin" ;;
        2) prompt_custom_path "Enter path" "$DEFAULT_TOOL_BIN"; TOOL_BIN_DIR="$CUSTOM_PATH" ;;
    esac
}

confirm_install() {
    echo ""
    echo -e "  ${BOLD}Summary:${NC}"
    echo -e "    Binary:     ${BIN_DIR}/fpm"
    echo -e "    Cache:      ${CACHE_DIR}"
    echo -e "    Data:       ${DATA_DIR}"
    echo -e "    Config:     ${CONFIG_DIR}"
    echo -e "    Tool bin:   ${TOOL_BIN_DIR}"

    NEEDS_SUDO=false
    if [ ! -w "$BIN_DIR" ] && [ -d "$BIN_DIR" ]; then
        NEEDS_SUDO=true
        echo -e "    ${YELLOW}Note: sudo required for ${BIN_DIR}${NC}"
    fi

    echo ""
    select_option "Confirm installation?" \
        "Install with these settings" \
        "Go with defaults (recommended)" \
        "Abort"
    case $SELECTED_INDEX in
        0) return 0 ;;  # proceed
        1) set_defaults; return 0 ;;  # reset to defaults
        2) echo -e "\n  ${DIM}Installation aborted.${NC}"; exit 0 ;;
    esac
}

# ─────────────────────────────────────────────────────────────────────────────
# Download and Install
# ─────────────────────────────────────────────────────────────────────────────

download_and_install() {
    echo ""

    # Get latest version
    step "Checking latest version..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/' || echo "")

    if [ -z "$VERSION" ]; then
        VERSION="latest"
        DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/fpm-${OS}-${ARCH}"
    else
        DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/fpm-${VERSION}-${OS}-${ARCH}"
    fi

    # Download
    step "Downloading fpm ${VERSION} for ${PLATFORM}..."
    TEMP_FILE=$(mktemp)
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE" 2>/dev/null; then
        # Fallback URL
        DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/fpm-${OS}-${ARCH}"
        if ! curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE" 2>/dev/null; then
            error "Download failed. Check your network connection."
            rm -f "$TEMP_FILE"
            exit 1
        fi
    fi

    chmod +x "$TEMP_FILE"
    local size=$(du -h "$TEMP_FILE" | awk '{print $1}')
    info "Downloaded (${size})"

    # Create directories
    mkdir -p "$CACHE_DIR" "$DATA_DIR" "$CONFIG_DIR" "$TOOL_BIN_DIR" 2>/dev/null || true
    mkdir -p "$BIN_DIR" 2>/dev/null || true

    # Install binary
    if [ -w "$BIN_DIR" ]; then
        mv "$TEMP_FILE" "${BIN_DIR}/fpm"
    else
        echo -e "  ${YELLOW}●${NC} Need elevated permissions for ${BIN_DIR}"
        sudo mv "$TEMP_FILE" "${BIN_DIR}/fpm"
    fi
    info "Installed to ${BIN_DIR}/fpm"

    # Create directories
    info "Created cache directory"
    info "Created data directory"

    # Write install receipt
    mkdir -p "$CONFIG_DIR"
    cat > "${CONFIG_DIR}/install-receipt.json" << EOF
{
  "version": "${VERSION}",
  "installed_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "platform": "${PLATFORM}",
  "bin_dir": "${BIN_DIR}",
  "cache_dir": "${CACHE_DIR}",
  "data_dir": "${DATA_DIR}",
  "config_dir": "${CONFIG_DIR}",
  "tool_bin_dir": "${TOOL_BIN_DIR}",
  "shell_profile": "${SHELL_PROFILE}"
}
EOF

    # Write user config if custom paths were chosen
    if [ "$CACHE_DIR" != "$DEFAULT_CACHE_DIR" ] || [ "$DATA_DIR" != "$DEFAULT_DATA_DIR" ]; then
        cat > "${CONFIG_DIR}/config.toml" << EOF
# fpm user configuration (generated by installer)

[cache]
dir = "${CACHE_DIR}"
EOF
        info "Created config with custom paths"
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# PATH Setup
# ─────────────────────────────────────────────────────────────────────────────

setup_path() {
    # Check if tool bin dir is already in PATH
    if echo "$PATH" | tr ':' '\n' | grep -qx "$TOOL_BIN_DIR"; then
        info "Tool bin already in PATH"
        return
    fi

    # Also check if bin dir needs PATH
    local needs_path=false
    if ! echo "$PATH" | tr ':' '\n' | grep -qx "$BIN_DIR"; then
        needs_path=true
    fi
    if ! echo "$PATH" | tr ':' '\n' | grep -qx "$TOOL_BIN_DIR"; then
        needs_path=true
    fi

    if [ "$needs_path" = false ]; then
        return
    fi

    local path_line=""
    case "$SHELL_NAME" in
        fish)
            path_line="set -gx PATH \"${TOOL_BIN_DIR}\" \$PATH"
            ;;
        *)
            path_line="export PATH=\"${TOOL_BIN_DIR}:\$PATH\""
            ;;
    esac

    if [ "$NONINTERACTIVE" = true ]; then
        echo "$path_line" >> "$SHELL_PROFILE"
        info "Added to PATH in ${SHELL_PROFILE}"
        return
    fi

    select_option "Add ${TOOL_BIN_DIR} to PATH?" \
        "Yes (modify ${SHELL_PROFILE})" \
        "No (I'll do it manually)"
    if [ $SELECTED_INDEX -eq 0 ]; then
        echo "" >> "$SHELL_PROFILE"
        echo "# fpm" >> "$SHELL_PROFILE"
        echo "$path_line" >> "$SHELL_PROFILE"
        info "Added to PATH in ${SHELL_PROFILE}"
    else
        warn "Add manually: ${path_line}"
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

main() {
    detect_platform
    detect_shell
    set_defaults

    banner

    echo -e "  Welcome to ${BOLD}fpm${NC}!"
    echo ""
    echo -e "  This will install fpm and configure directories for package"
    echo -e "  caching, Python version management, and tool isolation."

    if [ "$NONINTERACTIVE" = true ]; then
        show_defaults
        echo ""
        info "Non-interactive mode — using defaults"
        download_and_install
        setup_path
    else
        select_option "Installation type" \
            "Default (recommended)" \
            "Custom (choose paths)"

        if [ $SELECTED_INDEX -eq 0 ]; then
            show_defaults
            select_option "Proceed with installation?" \
                "Yes (recommended)" \
                "Abort"
            if [ $SELECTED_INDEX -eq 1 ]; then
                echo -e "\n  ${DIM}Installation aborted.${NC}"
                exit 0
            fi
        else
            run_custom_mode
            confirm_install
        fi

        download_and_install
        setup_path
    fi

    # Final message
    echo ""
    echo -e "  ${GREEN}${BOLD}fpm is ready!${NC} 🚀"
    echo ""
    echo -e "  ${DIM}Run:${NC}  fpm --help"
    echo -e "  ${DIM}Or:${NC}   fpm init myproject"
    echo ""

    # Hint about sourcing
    if [ "$SHELL_NAME" = "fish" ]; then
        echo -e "  ${DIM}To use immediately: source ${SHELL_PROFILE}${NC}"
    else
        echo -e "  ${DIM}To use immediately: source ${SHELL_PROFILE}${NC}"
    fi
    echo ""
}

main "$@"
