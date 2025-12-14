#!/bin/sh

# Configuration
REPO="npgy/gifthing"                    # Change to your repo
PROGRAM_NAME="gifthing"             # Name of your program
INSTALL_DIR="/opt/gifthing"         # Where to install
VERSION_FILE="$HOME/.${PROGRAM_NAME}_version"  # Track current version
BINARY_NAME="$PROGRAM_NAME"          # Name of the binary in releases
ASSET_PATTERN="linux_arm64"          # Pattern to match release asset

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Get current installed version
get_current_version() {
    if [[ -f "$VERSION_FILE" ]]; then
        cat "$VERSION_FILE"
    else
        echo "none"
    fi
}

# Get latest version from GitHub
get_latest_version() {
    local latest_tag
    latest_tag=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | \
                 jq -r '.tag_name // empty')

    if [[ -z "$latest_tag" ]]; then
        # Fallback to tags if no releases
        latest_tag=$(curl -s "https://api.github.com/repos/$REPO/tags" | \
                     jq -r '.[0].name // empty')
    fi

    echo "$latest_tag"
}

# Compare versions (assumes semantic versioning)
version_gt() {
    # Remove 'v' prefix if present
    local ver1="${1#v}"
    local ver2="${2#v}"

    # Simple version comparison
    printf '%s\n%s\n' "$ver2" "$ver1" | sort -V -c
}

# Download and extract release
download_release() {
    local version="$1"
    local temp_dir
    temp_dir=$(mktemp -d)

    log "Downloading $PROGRAM_NAME $version..."

    # Get download URL for the asset
    local download_url
    download_url=$(curl -s "https://api.github.com/repos/$REPO/releases/tags/$version" | \
                   jq -r ".assets[] | select(.name | contains(\"$ASSET_PATTERN\")) | .browser_download_url")

    if [[ -z "$download_url" ]]; then
        error "No matching asset found for pattern: $ASSET_PATTERN"
        return 1
    fi

    log "Download URL: $download_url"

    # Download the file
    local filename
    filename=$(basename "$download_url")

    if ! curl -L -o "$temp_dir/$filename" "$download_url"; then
        error "Failed to download $download_url"
        rm -rf "$temp_dir"
        return 1
    fi

    # Extract if it's an archive
    cd "$temp_dir" || exit 1

    case "$filename" in
        *.tar.gz|*.tgz)
            tar -xzf "$filename"
            ;;
        *.tar.bz2)
            tar -xjf "$filename"
            ;;
        *.zip)
            unzip -q "$filename"
            ;;
        *)
            # Assume it's a binary
            chmod +x "$filename"
            mv "$filename" "$BINARY_NAME"
            ;;
    esac

    # Find the binary
    local binary_path
    if [[ -f "$BINARY_NAME" ]]; then
        binary_path="$BINARY_NAME"
    else
        # Look for binary in subdirectories
        binary_path=$(find . -name "$BINARY_NAME" -type f -executable | head -1)
    fi

    if [[ -z "$binary_path" ]]; then
        error "Binary '$BINARY_NAME' not found in downloaded files"
        rm -rf "$temp_dir"
        return 1
    fi

    # Install the binary
    log "Installing to $INSTALL_DIR/$BINARY_NAME..."

    cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    # Cleanup
    rm -rf "$temp_dir"

    success "Successfully installed $PROGRAM_NAME $version"
    return 0
}

# Update version file
update_version_file() {
    echo "$1" > "$VERSION_FILE"
}

# Main update logic
main() {
    log "Checking for updates to $PROGRAM_NAME..."

    # Check if jq is installed
    if ! command -v jq &> /dev/null; then
        error "jq is required but not installed. Please install jq first."
        exit 1
    fi

    # Check if curl is installed
    if ! command -v curl &> /dev/null; then
        error "curl is required but not installed. Please install curl first."
        exit 1
    fi

    local current_version
    current_version=$(get_current_version)
    log "Current version: $current_version"

    local latest_version
    latest_version=$(get_latest_version)

    if [[ -z "$latest_version" ]]; then
        error "Could not determine latest version from GitHub"
        exit 1
    fi

    log "Latest version: $latest_version"

    # Compare versions
    if [[ "$current_version" == "$latest_version" ]]; then
        success "$PROGRAM_NAME is already up to date ($current_version)"
        exit 0
    elif [[ "$current_version" != "none" ]] && ! version_gt "$latest_version" "$current_version"; then
        warn "Current version ($current_version) is newer than or equal to latest ($latest_version)"
        if [[ "$1" != "--force" ]]; then
            log "Use --force to downgrade/reinstall"
            exit 0
        fi
    fi

    log "Update available: $current_version -> $latest_version"

    # Ask for confirmation unless --yes flag is passed
    if [[ "$1" != "--yes" && "$1" != "-y" && "$2" != "--yes" && "$2" != "-y" ]]; then
        read -p "Do you want to update? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log "Update cancelled"
            exit 0
        fi
    fi

    # Stop service if running
    rc-service gifthing stop

    # Download and install
    if download_release "$latest_version"; then
        update_version_file "$latest_version"
        success "Updated $PROGRAM_NAME from $current_version to $latest_version"

        # Verify installation
        if command -v "$BINARY_NAME" &> /dev/null; then
            log "Verification: $("$BINARY_NAME" --version 2>/dev/null || echo "Binary installed successfully")"
        fi

        rc-service gifthing start
    else
        error "Failed to update $PROGRAM_NAME"
        rc-service gifthing start
        exit 1
    fi
}

# Parse command line arguments
case "$1" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo "Update $PROGRAM_NAME from GitHub releases"
        echo ""
        echo "Options:"
        echo "  --yes, -y    Skip confirmation prompt"
        echo "  --force      Force reinstall even if version is same or newer"
        echo "  --check      Only check for updates, don't install"
        echo "  --help, -h   Show this help message"
        exit 0
        ;;
    --check)
        current_version=$(get_current_version)
        latest_version=$(get_latest_version)
        echo "Current: $current_version"
        echo "Latest:  $latest_version"
        if [[ "$current_version" == "$latest_version" ]]; then
            echo "Status: Up to date"
        else
            echo "Status: Update available"
        fi
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
