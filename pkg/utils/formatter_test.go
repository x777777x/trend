package utils

import (
	"testing"
)

func TestParseSizeToBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		hasError bool
	}{
		{"100", 100, false},
		{"100B", 100, false},
		{"1KB", 1024, false},
		{"2.5 MB", 2.5 * 1024 * 1024, false},
		{" 10gb ", 10 * 1024 * 1024 * 1024, false},
		{"1.2TB", 1.2 * 1024 * 1024 * 1024 * 1024, false},
		{"", 0, false},
		{"invalid", 0, true},
		{"100ZZZ", 0, true},
	}

	for _, tc := range tests {
		val, err := ParseSizeToBytes(tc.input)
		if tc.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", tc.input, err)
			}
			if val != tc.expected {
				t.Errorf("For input %q expected %f, got %f", tc.input, tc.expected, val)
			}
		}
	}
}
