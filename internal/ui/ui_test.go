package ui

import (
	"strings"
	"testing"
)

func TestBanner(t *testing.T) {
	b := Banner()
	if b == "" {
		t.Fatal("Banner() is empty")
	}
	// Trimmed of surrounding blank lines.
	if strings.HasPrefix(b, "\n") || strings.HasSuffix(b, "\n") {
		t.Error("Banner() should be trimmed of leading/trailing newlines")
	}
	// Carries the wordmark tagline.
	if !strings.Contains(b, "Further Reading") {
		t.Errorf("Banner() missing tagline; got:\n%s", b)
	}
}
