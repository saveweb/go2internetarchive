package metadata

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
)

func uriEscape(s string) string {
	return fmt.Sprintf("uri(%s)", url.QueryEscape(s))
}

// Copy from Golang's xml package
func isInCharacterRange(r rune) (inrange bool) {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

func IsLegalXMLChar(r rune) bool {
	return isInCharacterRange(r)
}

func ReplaceIllegalXMLChars(s string) (string, bool) {
	var b strings.Builder
	replaced := false
	for _, r := range s {
		if isInCharacterRange(r) {
			b.WriteRune(r)
		} else {
			replaced = true
			slog.Warn("An illegal XML character was replaced with U+FFFD.", "char", fmt.Sprintf("%U", r))
			b.Write([]byte("\uFFFD"))
		}
	}
	return b.String(), replaced
}
