#!/usr/bin/env bash

# Exit on error
set -e

# Define paths
INSTALL_DIR="$HOME/.local/bin/waybar-basecamp"
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
BINARY_NAME="waybar-basecamp"
SERVICE_NAME="waybar-basecamp.service"
TIMER_NAME="waybar-basecamp.timer"
REPO="jturmel/waybar-basecamp"

echo "Installing waybar-basecamp..."

# Function to download from GitHub release
download_from_release() {
    local file=$1
    if ! command -v curl >/dev/null 2>&1; then
        echo "Error: curl is not installed. Cannot download missing files."
        exit 1
    fi
    echo "Downloading $file from latest release..."
    curl -sL --fail "https://github.com/$REPO/releases/latest/download/$file" -o "$file" || {
        echo "Error: Failed to download $file. It might not be available in the latest release yet."
        exit 1
    }
}

# 1. Setup directory in ~/.local/bin
echo "Creating directory $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"

# 2. Check and pull missing files
for file in "$BINARY_NAME" "$SERVICE_NAME" "$TIMER_NAME"; do
    if [ ! -f "$file" ]; then
        download_from_release "$file"
    fi
done

# Check if the binary exists now
if [ ! -f "$BINARY_NAME" ]; then
    echo "Error: $BINARY_NAME not found and failed to download."
    exit 1
fi

# 3. Copy the binary
echo "Copying binary to $INSTALL_DIR..."
cp "$BINARY_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# 4. Run the setup command only if configuration doesn't exist
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/waybar-basecamp"
CONFIG_FILE="$CONFIG_DIR/config.json"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Configuration not found. Running setup..."
    "$INSTALL_DIR/$BINARY_NAME" setup
else
    echo "Configuration already exists at $CONFIG_FILE. Skipping setup."
    echo "Run '$INSTALL_DIR/$BINARY_NAME setup' manually if you need to reconfigure."
fi

# 5. Copy configuration files (service and timer)
echo "Setting up systemd user services..."
mkdir -p "$SYSTEMD_USER_DIR"
cp "$SERVICE_NAME" "$SYSTEMD_USER_DIR/"
cp "$TIMER_NAME" "$SYSTEMD_USER_DIR/"

# 6. Reload systemd user daemon
echo "Reloading systemd user daemon..."
systemctl --user daemon-reload

# 7. Enable and start the timer
echo "Enabling and starting $TIMER_NAME..."
systemctl --user enable "$TIMER_NAME"
systemctl --user start "$TIMER_NAME"

echo "Installation complete!"
echo "Note: Make sure ~/.local/bin is in your PATH if you want to run the binary manually."
