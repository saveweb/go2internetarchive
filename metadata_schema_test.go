package go2internetarchive

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

func Test_uriEscape(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"hello world", "uri(hello+world)"},
		{"hell\no-world", "uri(hell%0Ao-world)"},
		{"hello_world", "uri(hello_world)"},
		{"hello+world", "uri(hello%2Bworld)"},
		{"hello%world", "uri(hello%25world)"},
		{"hello👋", "uri(hello%F0%9F%91%8B)"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := uriEscape(tt.s); got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
