package theme

import (
	"slices"
	"testing"
)

// TestNames covers Names(): cycle order plus the auto/ascii pseudo-themes.
func TestNames(t *testing.T) {
	names := Names()
	if len(names) != len(CycleOrder)+2 {
		t.Fatalf("Names() len = %d, want %d", len(names), len(CycleOrder)+2)
	}
	if !slices.Contains(names, "auto") || !slices.Contains(names, "ascii") {
		t.Errorf("Names() missing pseudo-themes: %v", names)
	}
	// First entries must match the cycle order.
	for i, c := range CycleOrder {
		if names[i] != c {
			t.Errorf("Names()[%d] = %q, want %q", i, names[i], c)
		}
	}
}
