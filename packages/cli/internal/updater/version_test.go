package updater

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected int
		wantErr  bool
	}{
		{"older patch", "1.0.0", "1.0.1", -1, false},
		{"older minor", "1.0.0", "1.1.0", -1, false},
		{"older major", "1.0.0", "2.0.0", -1, false},
		{"equal", "1.2.3", "1.2.3", 0, false},
		{"newer", "1.1.0", "1.0.0", 1, false},
		{"v prefix current", "v1.0.0", "1.0.1", -1, false},
		{"v prefix latest", "1.0.0", "v1.0.1", -1, false},
		{"v prefix both", "v1.0.0", "v1.0.1", -1, false},
		{"prerelease less than release", "1.0.0-beta", "1.0.0", -1, false},
		{"prerelease comparison", "1.0.0-alpha", "1.0.0-beta", -1, false},
		{"invalid current", "notaversion", "1.0.0", 0, true},
		{"invalid latest", "1.0.0", "notaversion", 0, true},
		{"dev version", "dev", "1.0.0", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareVersions(tt.current, tt.latest)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.current, tt.latest, result, tt.expected)
			}
		})
	}
}

func TestIsUpdateAvailable(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{"update available", "1.0.0", "1.1.0", true},
		{"on latest", "1.1.0", "1.1.0", false},
		{"ahead of latest", "1.2.0", "1.1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsUpdateAvailable(tt.current, tt.latest)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("IsUpdateAvailable(%q, %q) = %v, want %v", tt.current, tt.latest, result, tt.expected)
			}
		})
	}
}
