// Package ui holds shared presentation assets for the CLI.
package ui

import (
	_ "embed"
	"strings"
)

//go:embed banner.txt
var bannerRaw string

//go:embed logo.png
var logoPNG []byte

// Banner is the fur ASCII banner with surrounding blank lines trimmed, ready
// to print above command output.
func Banner() string {
	return strings.Trim(bannerRaw, "\n")
}

// LogoPNG returns the embedded fur logo as PNG bytes, for inline rendering in
// terminals that support an image protocol (Kitty/iTerm2). Callers fall back to
// Banner() when no protocol is available.
func LogoPNG() []byte {
	return logoPNG
}
