<h1 align="center">
  ⚡ zugbar
</h1>

<p align="center">
  <img src="ScreenRecording_05-01-2026 07-51-28_1.gif" alt="Zugbar network monitor demo" width="600"/>
</p>

<p align="center">
  <strong>Network monitoring that flows like electricity</strong><br>
  Stackable Waybar widget with iOS-style cycling and sparkline visualization
</p>

<p align="center">
  Built for CachyOS & Hyprland ⚡🇨🇭
</p>

---

# zugbar

Stackable network monitoring widget for waybar with iOS-style cycling.

Click through multiple network targets with beautiful sparkline visualization.

---

## Features

- **Click-to-cycle** - Click to cycle through targets (iOS widget style)
- **DPI-independent sparklines** - Uses 21-level block characters, no canvas/resize issues
- **Color thresholds** - Matches the Dracula/Drunk theme
- **waybar-ready** - JSON output with tooltips
- **History tracking** - Stores latency data for sparkline visualization

---

## Targets

Default targets configured in `main.go`:
| Name | Host | Label |
|------|------|-------|
| BadBitch | 10.0.0.15 | BB |
| Router | 10.0.0.1 | RTR |
| Proxmox | 10.0.0.2 | PVE |

Edit the `targets` slice in `main.go` to customize.

---

## Installation

```bash
# Clone and build
git clone https://github.com/richknowles/zugbar.git
cd zugbar
go build -o zugbar main.go

# Copy to your waybar scripts folder
cp zugbar ~/.config/waybar/scripts/
chmod +x ~/.config/waybar/scripts/zugbar
```

---

## waybar Configuration

Add to your waybar `config`:

```json
"custom/zugbar": {
  "exec": "~/.config/waybar/scripts/zugbar",
  "return": 0,
  "interval": 2,
  "on-click": "~/.config/waybar/scripts/zugbar cycle",
  "tooltip": true
}
```

### CSS Styling (optional)

Add to your waybar `style.css`:

```css
.zugbar-good {
  color: #50fa7b;
}
.zugbar-medium {
  color: #ffb86c;
}
.zugbar-slow {
  color: #ff79c6;
}
.zugbar-offline {
  color: #ff5555;
}
```

---

## Usage

```bash
# Run the widget
./zugbar

# Cycle to next target
./zugbar cycle

# Check current target
cat /tmp/zugbar_state

# View history
cat /tmp/zugbar_history
```

---

## Sparkline Algorithm

```
// 21-level DPI blocks (no canvas, no resize math)
const BLOCKS = ['░','▁','▂','▃','▄','▅','▆','▇','█','█▁','█▂','█▃','█▄','█▅','█▆','█▇','██','██▁','██▂','██▃','███'];

// Color thresholds (latency in microseconds)
idle:     #6272a4  (< 1ms)
// light:    #8be9fd  (< 10ms)
// moderate: #50fa7b  (< 100ms)
// heavy:    #ffb86c  (< 500ms)
// extreme: #ff5555  (>= 500ms)
```

---

## Architecture

```
main.go       - All logic in single file
  ├── config  - Targets, colors, blocks
  ├── state   - Load/save cycle index
  ├── ping    - Shell ping with regex parsing
  ├── sparkline - Block + color mapping
  └── output   - waybar JSON format
```

---

## Files

- `main.go` - Go source (compile with `go build`)
- `zugbar` - Compiled binary
- `config.json` - Example waybar config

---

## License

MIT © 2026 Rich Knowles

---

## Author

**Rich Knowles**  
rich@itwerks.net  
https://github.com/richknowles/zugbar
