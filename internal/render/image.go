package render

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImageProtocol identifies the terminal image protocol to use.
type ImageProtocol int

const (
	ImageProtocolNone ImageProtocol = iota
	ImageProtocolKitty
	ImageProtocolITerm2
)

// DetectImageProtocol returns the best available terminal image protocol.
func DetectImageProtocol() ImageProtocol {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	if strings.Contains(term, "kitty") || termProgram == "kitty" || termProgram == "ghostty" {
		return ImageProtocolKitty
	}
	if termProgram == "iTerm.app" || termProgram == "WezTerm" {
		return ImageProtocolITerm2
	}
	if os.Getenv("LC_TERMINAL") == "iTerm2" {
		return ImageProtocolITerm2
	}
	return ImageProtocolNone
}

// RenderImageInline outputs an image using the given terminal protocol.
// Only safe outside alt-screen (e.g., `lookit cat`), NOT inside Bubble Tea.
func RenderImageInline(path string, protocol ImageProtocol) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading image: %w", err)
	}

	switch protocol {
	case ImageProtocolKitty:
		return renderKitty(data), nil
	case ImageProtocolITerm2:
		return renderITerm2(data, path), nil
	default:
		return fmt.Sprintf("[image: %s — no inline image protocol detected]\n", filepath.Base(path)), nil
	}
}

func renderKitty(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var b strings.Builder
	for i := 0; i < len(encoded); i += 4096 {
		end := i + 4096
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		more := 1
		if end >= len(encoded) {
			more = 0
		}
		if i == 0 {
			fmt.Fprintf(&b, "\033_Ga=T,f=100,m=%d;%s\033\\", more, chunk)
		} else {
			fmt.Fprintf(&b, "\033_Gm=%d;%s\033\\", more, chunk)
		}
	}
	b.WriteString("\n")
	return b.String()
}

func renderITerm2(data []byte, path string) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	name := base64.StdEncoding.EncodeToString([]byte(filepath.Base(path)))
	return fmt.Sprintf("\033]1337;File=name=%s;inline=1;width=auto;preserveAspectRatio=1:%s\a\n",
		name, encoded)
}
