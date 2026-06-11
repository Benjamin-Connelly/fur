#!/usr/bin/env bash
# Regenerate the theme gallery images in docs/themes/img/.
#
# Renders sample.md with `fur cat --theme <name>` for each gallery theme,
# captures it with charmbracelet/freeze (SVG), then rasterizes to PNG with
# headless Chrome. freeze's --background is set to each theme's base color so
# light themes render on a light window (fur themes set foreground only and
# assume the terminal background matches).
#
# Requirements: fur (on PATH), freeze (go install github.com/charmbracelet/freeze@v0.2.2),
# google-chrome.
#
# Usage: bash docs/themes/generate.sh

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

CHROME="${CHROME:-google-chrome}"
SAMPLE="sample.md"
OUT="img"
mkdir -p "$OUT"

# Gallery themes -> base background color (palette Bg). Keep in sync with
# internal/theme/palettes.go.
THEMES=(
    "catppuccin-mocha #1e1e2e"
    "catppuccin-latte #eff1f5"
    "dracula          #282a36"
    "nord             #2e3440"
    "gruvbox          #282828"
    "tokyonight-night #1a1b26"
)

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

for entry in "${THEMES[@]}"; do
    read -r theme bg <<<"$entry"
    svg="$tmp/$theme.svg"
    freeze --execute "fur cat --theme $theme $SAMPLE" \
        --background "$bg" --window --padding 20 --border.radius 8 \
        --output "$svg"
    w=$(grep -o 'width="[0-9.]*"' "$svg" | head -1 | grep -o '[0-9]*' | head -1)
    h=$(grep -o 'height="[0-9.]*"' "$svg" | head -1 | grep -o '[0-9]*' | head -1)
    "$CHROME" --headless=new --disable-gpu --no-sandbox \
        --force-device-scale-factor=2 --hide-scrollbars \
        --default-background-color=00000000 \
        --window-size="${w},${h}" \
        --screenshot="$OUT/$theme.png" "file://$svg" >/dev/null 2>&1
    echo "wrote $OUT/$theme.png"
done
