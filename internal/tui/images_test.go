package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".png", true},
		{".PNG", true},
		{".jpg", true},
		{".jpeg", true},
		{".gif", true},
		{".bmp", true},
		{".webp", true},
		{".svg", true},
		{".ico", true},
		{".go", false},
		{".md", false},
		{".txt", false},
		{"", false},
		{".pdf", false},
	}
	for _, tt := range tests {
		got := IsImageFile(tt.ext)
		if got != tt.want {
			t.Errorf("IsImageFile(%q) = %v, want %v", tt.ext, got, tt.want)
		}
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
	}
	for _, tt := range tests {
		got := formatFileSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatFileSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestImageRenderer_Render(t *testing.T) {
	dir := t.TempDir()

	// Create a minimal 1x1 PNG (valid image header)
	// This is the smallest valid PNG: 1x1 pixel, 8-bit RGBA
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, // 8-bit RGB
		0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54, // IDAT chunk
		0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, // compressed data
		0xE2, 0x21, 0xBC, 0x33, // CRC
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, // IEND chunk
		0xAE, 0x42, 0x60, 0x82, // CRC
	}
	imgPath := filepath.Join(dir, "test.png")
	if err := os.WriteFile(imgPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewImageRenderer()
	result := r.Render(imgPath)

	if !strings.Contains(result, "test.png") {
		t.Error("should contain filename")
	}
	if !strings.Contains(result, "PNG") {
		t.Error("should contain file type")
	}
	if !strings.Contains(result, "B") {
		t.Error("should contain file size")
	}
	if !strings.Contains(result, "Press 'e'") {
		t.Error("should contain open hint")
	}
}

// TestImageRenderer_Render_WebPDims guards the webp/bmp decoder registration:
// without the blank x/image imports, image.DecodeConfig fails on webp and the
// info card silently drops the dimensions line.
func TestImageRenderer_Render_WebPDims(t *testing.T) {
	// A committed demo asset; skip if the demo tree has been relocated.
	webp := filepath.Join("..", "..", "docs", "demo", "millions-of-cats", "images", "cover.webp")
	if _, err := os.Stat(webp); err != nil {
		t.Skipf("demo webp not present: %v", err)
	}
	result := NewImageRenderer().Render(webp)
	if !strings.Contains(result, "WEBP") {
		t.Errorf("should report WEBP type, got:\n%s", result)
	}
	if !strings.Contains(result, "Dims:") {
		t.Errorf("webp dimensions should decode (decoder registered), got:\n%s", result)
	}
}

func TestImageRenderer_Render_NotFound(t *testing.T) {
	r := NewImageRenderer()
	result := r.Render("/nonexistent/image.png")
	if !strings.Contains(result, "cannot read file") {
		t.Errorf("should indicate file error, got %q", result)
	}
}

func TestImageRenderer_Render_NonImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")
	os.WriteFile(path, []byte("not an image"), 0o644)

	r := NewImageRenderer()
	result := r.Render(path)
	// Should still render info card (just no dimensions)
	if !strings.Contains(result, "data.bin") {
		t.Error("should contain filename even for non-image")
	}
}
