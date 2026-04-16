package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	// Register image decoders for non-PNG formats
	_ "image/gif"
	_ "image/jpeg"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
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
// Only safe outside alt-screen (e.g., `fur cat`), NOT inside Bubble Tea.
// An optional afero.Fs can be provided; if nil, the OS filesystem is used.
func RenderImageInline(path string, protocol ImageProtocol, fsys ...afero.Fs) (string, error) {
	var fs afero.Fs = afero.NewOsFs()
	if len(fsys) > 0 && fsys[0] != nil {
		fs = fsys[0]
	}
	data, err := afero.ReadFile(fs, path)
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

// toPNG converts image data to PNG if it isn't already.
// Kitty protocol f=100 requires PNG format.
func toPNG(data []byte) []byte {
	if len(data) >= 8 && string(data[:4]) == "\x89PNG" {
		return data
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data // can't decode — send raw, terminal may reject
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return data
	}
	return buf.Bytes()
}

func renderKitty(data []byte) string {
	data = toPNG(data)
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
