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
