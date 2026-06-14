// Package ui holds shared presentation assets for the CLI.
package ui

import (
	_ "embed"
	"strings"
)

//go:embed banner.txt
var bannerRaw string

// Banner is the fur ASCII banner with surrounding blank lines trimmed, ready
// to print above command output.
func Banner() string {
	return strings.Trim(bannerRaw, "\n")
}
