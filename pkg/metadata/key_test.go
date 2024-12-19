package metadata

import (
	"strings"
	"testing"
)

func Test_IsValidKey(t *testing.T) {
	tests := []struct {
		key  string
		errW error
	}{
		{"", KeyEmptyError},
		{strings.Repeat("a", 257), KeyTooLongError},
		{"a--bc", KeyDoubleHyphensError},
		{"a-_b", KeyHyphenWithUnderscoreError},
		{"a_-b", KeyHyphenWithUnderscoreError},
		{"123abc", KeyInvalidStartError},
		{"ABC123", KeyuUpcaseError},
		{"abc___123", nil},
		{"avr-123dc_adsd923-sd2.312-123.123_123", nil},
		{"a.b.c", nil},
		{"yzqzss@saveweb.org", KeyInllegalCharError},
		{"saveweb@saveweb.org", KeyInllegalCharError},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if err := isValidKey(tt.key); err != tt.errW {
				t.Fatalf("want %v, got %v", tt.errW, err)
			}
		})
	}

}
