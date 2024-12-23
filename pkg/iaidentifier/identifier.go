package iaidentifier

// Each item at Internet Archive has an identifier. An identifier is composed of a unique combination of alphanumeric characters (limited to ASCII), underscores (_), dashes (-), or periods (.). The first character of an identifier must be alphanumeric (e.g. it cannot start out with an underscore, dash, or period). The maximum length of an identifier is 100 characters, but we generally recommend that identifiers be between 5 and 80 characters in length.
func IsValidIdentifier(identifier string) bool {
	if len(identifier) == 0 {
		return false
	}
	if len(identifier) > 100 {
		return false
	}

	if identifier[0] < 'a' || identifier[0] > 'z' {
		return false
	}

	for _, c := range identifier {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '_' || c == '-' || c == '.' {
			continue
		}
		return false
	}

	return true
}
