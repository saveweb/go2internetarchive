package metadata

import (
	"testing"
	"unicode/utf8"
)

func Test_uriEscape(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"hello world", "uri(hello%20world)"},
		{"hello+world", "hello+world"},
		{"hello_world", "hello_world"},
		{"hell\no-world", "uri(hell%0Ao-world)"},
		{"hello%world", "uri(hello%25world)"},
		{"hello👋", "uri(hello%F0%9F%91%8B)"},
		{"hello This+is+meta1, !@#$%^&*()_+{}|:\"<>? 你好👋", "uri(hello%20This+is+meta1%2C%20%21@%23$%25%5E&%2A%28%29_+%7B%7D%7C:%22%3C%3E%3F%20%E4%BD%A0%E5%A5%BD%F0%9F%91%8B)"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := uriEscape(tt.s); got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func Test_IsLegalXMLChar(t *testing.T) {
	invalid := []rune{
		utf8.MaxRune + 1,
		0xD800, // surrogate min
		0xDFFF, // surrogate max
		-1,
	}
	for _, r := range invalid {
		if IsLegalXMLChar(r) {
			t.Errorf("rune %U considered valid", r)
		}
	}
}

func Test_ReplaceIllegalXMLChars(t *testing.T) {
	var characterTests = []struct {
		input string
		want  string
	}{
		{"", ""},
		{"\x12<doc/>", "\uFFFD<doc/>"},
		{"<?xml version=\"1.0\"?>\x0b<doc/>", "<?xml version=\"1.0\"?>\uFFFD<doc/>"},
		{"\xef\xbf\xbe<doc/>", "\uFFFD<doc/>"},
		{"<?xml version=\"1.0\"?><doc>\r\n<hiya/>\x07<toots/></doc>", "<?xml version=\"1.0\"?><doc>\r\n<hiya/>\uFFFD<toots/></doc>"},
		{"\uFFFD", "\uFFFD"},
	}
	for _, tt := range characterTests {
		t.Run(tt.input, func(t *testing.T) {
			if got, _ := ReplaceIllegalXMLChars(tt.input); got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
