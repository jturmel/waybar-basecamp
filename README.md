# waybar-basecamp

A simple Go-based tool to show Basecamp notification counts in your Waybar. It uses your browser's cookies to authenticate with Basecamp and provides a JSON output compatible with Waybar's custom modules.

## Features

- **Automatic Authentication**: Leverages existing browser cookies (Chrome/Chromium-based).
- **Systemd Integration**: Runs as a background service with a timer to keep notification counts updated.
- **Waybar Compatible**: Outputs JSON in the format Waybar expects.
- **Easy Setup**: Interactive setup command to configure your account and browser profile.

## Prerequisites

- **Linux** (Tested on Wayland/Sway/Hyprland)
- **libsecret-1-dev**: Required for decrypting browser cookies on some systems.
  - Ubuntu/Debian: `sudo apt install libsecret-1-dev`
  - Fedora: `sudo dnf install libsecret-devel`
  - Arch: `sudo pacman -S libsecret`
- **curl**: Required for the automatic installation script.

## Installation

### Method 1: Quick Install (Recommended)

You can install `waybar-basecamp` by downloading and running the installation script directly:

```bash
curl -sL https://github.com/jturmel/waybar-basecamp/releases/latest/download/install.sh | bash
```

The script will:
1. Download the latest binary and systemd files.
2. Install them to `~/.local/bin/waybar-basecamp/`.
3. Run an interactive setup if no configuration exists.
4. Set up and start a systemd user timer.

### Method 2: From Source

If you have Go installed, you can build and install manually:

1. Clone the repository:
   ```bash
   git clone https://github.com/jturmel/waybar-basecamp.git
   cd waybar-basecamp
   ```
2. Build the binary:
   ```bash
   make build
   ```
3. Run the installation script:
   ```bash
   ./install.sh
   ```

## Configuration

During installation, you will be prompted for:
1. **Browser Profile**: The name of your Chrome/Chromium profile folder (e.g., `Default`, `Profile 1`).
2. **Account ID**: Your Basecamp account ID, found in the URL after logging in (e.g., `https://3.basecamp.com/1234567/...`).

You can re-run the setup anytime:
```bash
~/.local/bin/waybar-basecamp/waybar-basecamp setup
```

## Waybar Integration

Add the following to your Waybar configuration (usually `~/.config/waybar/config` or `config.jsonc`):

```jsonc
"custom/basecamp": {
    "format": "BC: {}",
    "return-type": "json",
    "exec": "cat /tmp/waybar_basecamp.json",
    "interval": 1,
    "on-click": "xdg-open https://3.basecamp.com/ACCOUNT_ID/", // Replace ACCOUNT_ID
    "signal": 8
}
```

Add it to your bar modules:
```jsonc
"modules-right": [
    ...
    "custom/basecamp",
    ...
],
```

### Styling (Optional)

You can style the module in `style.css`:

```css
#custom-basecamp.unread {
    color: #ffb86c;
}
#custom-basecamp.empty {
    color: #6272a4;
}
#custom-basecamp.error {
    color: #ff5555;
}
```

## How it Works

1. **`waybar-basecamp check`**: This command (triggered by systemd) reads your browser cookies, fetches the notification count from Basecamp's API, and writes the result to `/tmp/waybar_basecamp.json`.
2. **Systemd Timer**: Runs every minute to update the notification count.
3. **Waybar**: Reads the JSON file and displays the count. The tool sends a signal (`RTMIN+8`) to Waybar to refresh the module immediately after a check.
