#!/bin/bash
# zugbar installer for waybar
# Usage: ./install.sh [--themes-dir ~/.config/waybar/themes]

set -e

THEME_DIR="${HOME}/.config/waybar/themes"
SCRIPT_DIR="${HOME}/.config/waybar/scripts"

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --themes-dir)
            THEME_DIR="$2"
            shift 2
            ;;
        --scripts-dir)
            SCRIPT_DIR="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "🎯 Installing zugbar..."

# Create scripts dir if needed
mkdir -p "$SCRIPT_DIR"

# Get directory of this script
SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Copy binary
echo "📦 Copying zugbar binary..."
cp "$SCRIPT_PATH/zugbar" "$SCRIPT_DIR/zugbar"
chmod +x "$SCRIPT_DIR/zugbar"

# Add module to waybar if not exists
MODULE_BLOCK='
  // Zugbar - Stackable Network Cycler
  "custom/zugbar": {
    "exec": "~/.config/waybar/scripts/zugbar",
    "format": "{}",
    "return-type": "json",
    "interval": 2,
    "escape": false,
    "on-click": "~/.config/waybar/scripts/zugbar cycle",
    "tooltip-format": "Network Monitor - Click to cycle"
  },'

if ! grep -q 'custom/zugbar' "$HOME/.config/waybar/modules.json" 2>/dev/null; then
    echo "📝 Adding zugbar module to modules.json..."
    # Add after network-azucar section
    sed -i '/Network Activity"/a\'"$MODULE_BLOCK" "$HOME/.config/waybar/modules.json"
fi

# Create theme
echo "🎨 Creating zugbar theme..."
mkdir -p "$THEME_DIR/zugbar/default"

cat > "$THEME_DIR/zugbar/config" << 'EOF'
{
    "layer": "top",
    "margin-top": 8,
    "margin-bottom": 0,
    "margin-left": 8,
    "margin-right": 8,
    "spacing": 0,
    "include": [
        "~/.config/ml4w/settings/waybar-quicklinks.json",
        "~/.config/waybar/modules.json"
    ],
    "modules-left": [
        "custom/appmenu",
        "clock",
        "custom/net-traffic",
        "custom/zugbar",
        "custom/ai"
    ],
    "modules-center": [
        "hyprland/workspaces",
        "custom/empty"
    ],
    "modules-right": [
        "custom/updates",
        "pulseaudio",
        "bluetooth",
        "network",
        "battery",
        "group/hardware",
        "group/tools",
        "tray",
        "custom/notification",
        "custom/exit",
        "custom/ml4w-welcome"
    ]
}
EOF

cat > "$THEME_DIR/zugbar/default/config.sh" << 'EOF'
#!/bin/bash
theme_name="zugbar"
EOF

cat > "$THEME_DIR/zugbar/default/style.css" << 'EOF'
@import '../../../../../.config/waybar/colors.css';
@import '../../ml4w-glass/style.css';

.zugbar-good { color: #50fa7b; }
.zugbar-medium { color: #ffb86c; }
.zugbar-slow { color: #ff79c6; }
.zugbar-offline { color: #ff5555; }
EOF

chmod +x "$THEME_DIR/zugbar/default/config.sh"

echo ""
echo "✅ zugbar installed!"
echo ""
echo "To use the theme:"
echo "  1. Select 'zugbar' in waybar settings"
echo "  OR add 'custom/zugbar' to your existing theme's modules"
echo ""
echo "To add to an existing theme, add 'custom/zugbar' to modules-left or modules-right"
echo ""
echo "Run 'zugbar' to test, 'zugbar cycle' to switch targets"
echo ""
echo "Default targets: BadBitch → Router → Proxmox"
echo "Edit main.go to customize targets, then rebuild:"
echo "  go build -o zugbar main.go"