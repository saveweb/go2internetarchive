package iaidentifier

import "fmt"

// Each item at Internet Archive has an identifier. An identifier is composed of
// a unique combination of alphanumeric characters (limited to ASCII), underscores (_),
// dashes (-), or periods (.). The first character of an identifier must be
// alphanumeric (e.g. it cannot start out with an underscore, dash, or period).
//
// The maximum length of an identifier is 100 characters, but we generally
// recommend that identifiers be between 5 and 80 characters in length.

func IsValidIdentifier(identifier string) error {
	if len(identifier) == 0 {
		return fmt.Errorf("identifier is empty")
	}
	if len(identifier) > 100 {
		return fmt.Errorf("identifier is too long (max 100 characters)")
	}

	for idx, c := range identifier {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= 'A' && c <= 'Z' {
			continue
		}

		if idx == 0 {
			return fmt.Errorf("identifier cannot start with %c", c)
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '_' || c == '-' || c == '.' {
			continue
		}
		return fmt.Errorf("invalid character in identifier: %c", c)
	}

	return nil
}
