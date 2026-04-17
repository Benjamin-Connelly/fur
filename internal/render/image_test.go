package render

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding test PNG: %v", err)
	}
	return buf.Bytes()
}

func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 50}); err != nil {
		t.Fatalf("encoding test JPEG: %v", err)
	}
	return buf.Bytes()
}

func TestDetectImageProtocol(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want ImageProtocol
	}{
		{
			name: "kitty via TERM",
			env:  map[string]string{"TERM": "xterm-kitty", "TERM_PROGRAM": "", "LC_TERMINAL": ""},
			want: ImageProtocolKitty,
		},
		{
			name: "kitty via TERM_PROGRAM",
			env:  map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "kitty", "LC_TERMINAL": ""},
			want: ImageProtocolKitty,
		},
		{
			name: "ghostty uses kitty protocol",
			env:  map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "ghostty", "LC_TERMINAL": ""},
			want: ImageProtocolKitty,
		},
		{
			name: "iTerm via TERM_PROGRAM",
			env:  map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "iTerm.app", "LC_TERMINAL": ""},
			want: ImageProtocolITerm2,
		},
		{
			name: "WezTerm uses iTerm2 protocol",
			env:  map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "WezTerm", "LC_TERMINAL": ""},
			want: ImageProtocolITerm2,
		},
		{
			name: "iTerm via LC_TERMINAL",
			env:  map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "", "LC_TERMINAL": "iTerm2"},
			want: ImageProtocolITerm2,
		},
		{
			name: "no protocol detected",
			env:  map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "", "LC_TERMINAL": ""},
			want: ImageProtocolNone,
		},
		{
			name: "empty env",
			env:  map[string]string{"TERM": "", "TERM_PROGRAM": "", "LC_TERMINAL": ""},
			want: ImageProtocolNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got := DetectImageProtocol()
			if got != tt.want {
				t.Errorf("DetectImageProtocol() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestToPNG_AlreadyPNG(t *testing.T) {
	data := makePNG(t, 2, 2)
	result := toPNG(data)
	if !bytes.Equal(result, data) {
		t.Error("toPNG should return identical bytes for PNG input")
	}
}

func TestToPNG_ConvertsJPEG(t *testing.T) {
	jpegData := makeJPEG(t, 4, 4)
	result := toPNG(jpegData)
	if string(result[:4]) != "\x89PNG" {
		t.Error("toPNG should convert JPEG to PNG")
	}
	if bytes.Equal(result, jpegData) {
		t.Error("converted output should differ from JPEG input")
	}
}

func TestToPNG_InvalidData(t *testing.T) {
	garbage := []byte("not an image at all")
	result := toPNG(garbage)
	if !bytes.Equal(result, garbage) {
		t.Error("toPNG should return raw data when decoding fails")
	}
}

func TestRenderKitty(t *testing.T) {
	data := makePNG(t, 2, 2)
	result := renderKitty(data)

	if !strings.HasPrefix(result, "\033_Ga=T,f=100,") {
		t.Error("kitty output should start with APC graphics command")
	}
	if !strings.HasSuffix(result, "\033\\\n") {
		t.Error("kitty output should end with ST and newline")
	}
	if !strings.Contains(result, "m=0;") {
		t.Error("last chunk should have m=0 (no more data)")
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	if !strings.Contains(result, encoded) {
		t.Error("kitty output should contain base64-encoded image data")
	}
}

func TestRenderKitty_Chunking(t *testing.T) {
	// Need raw data whose base64 exceeds 4096 chars: ceil(4096 * 3/4) = 3073 bytes minimum.
	// Use a fake "PNG" by prepending the PNG magic bytes to enough filler data.
	fakeData := make([]byte, 4000)
	copy(fakeData, "\x89PNG")
	for i := 4; i < len(fakeData); i++ {
		fakeData[i] = byte(i % 251)
	}
	result := renderKitty(fakeData)

	encoded := base64.StdEncoding.EncodeToString(fakeData)
	if len(encoded) <= 4096 {
		t.Fatalf("test data too small: base64 len = %d", len(encoded))
	}

	chunks := strings.Count(result, "\033_G")
	if chunks < 2 {
		t.Errorf("expected multiple chunks, got %d", chunks)
	}
	if !strings.Contains(result, "m=1;") {
		t.Error("intermediate chunks should have m=1 (more data)")
	}
	if !strings.Contains(result, "m=0;") {
		t.Error("last chunk should have m=0")
	}
}

func TestRenderITerm2(t *testing.T) {
	data := makePNG(t, 2, 2)
	path := "/tmp/test-image.png"
	result := renderITerm2(data, path)

	if !strings.HasPrefix(result, "\033]1337;File=") {
		t.Error("iTerm2 output should start with OSC 1337 file sequence")
	}
	if !strings.HasSuffix(result, "\a\n") {
		t.Error("iTerm2 output should end with BEL and newline")
	}

	encodedName := base64.StdEncoding.EncodeToString([]byte(filepath.Base(path)))
	if !strings.Contains(result, "name="+encodedName) {
		t.Error("iTerm2 output should contain base64-encoded filename")
	}
	if !strings.Contains(result, "inline=1") {
		t.Error("iTerm2 output should set inline=1")
	}

	encodedData := base64.StdEncoding.EncodeToString(data)
	if !strings.Contains(result, ":"+encodedData) {
		t.Error("iTerm2 output should contain base64-encoded image data after colon")
	}
}

func TestRenderImageInline(t *testing.T) {
	pngData := makePNG(t, 2, 2)
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/img.png", pngData, 0644)

	tests := []struct {
		name     string
		protocol ImageProtocol
		contains string
	}{
		{"kitty", ImageProtocolKitty, "\033_Ga=T,f=100,"},
		{"iterm2", ImageProtocolITerm2, "\033]1337;File="},
		{"none", ImageProtocolNone, "[image: img.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderImageInline("/img.png", tt.protocol, fs)
			if err != nil {
				t.Fatalf("RenderImageInline: %v", err)
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected output to contain %q", tt.contains)
			}
		})
	}
}

func TestRenderImageInline_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := RenderImageInline("/nonexistent.png", ImageProtocolKitty, fs)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestRenderImageInline_NilFs(t *testing.T) {
	// Passing nil fs should fall back to OS filesystem; a missing file should error
	_, err := RenderImageInline("/definitely/not/a/real/path.png", ImageProtocolKitty, nil)
	if err == nil {
		t.Error("expected error reading from OS filesystem with bad path")
	}
}
