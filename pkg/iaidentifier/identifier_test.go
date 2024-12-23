package iaidentifier

import (
	"strings"
	"testing"
)

func Test_IsValidIdentifier(t *testing.T) {
	tests := []struct {
		identifier string
		want       bool
	}{
		{"", false},
		{"a", true},
		{"a-b", true},
		{"a_b", true},
		{"a.b,", false},
		{strings.Repeat("a", 101), false},
		{"1a", false},
		{"-a", false},
		{"_a", false},
	}
	for _, tt := range tests {
		t.Run(tt.identifier, func(t *testing.T) {
			if got := IsValidIdentifier(tt.identifier); got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
