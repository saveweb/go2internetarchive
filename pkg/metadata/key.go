package metadata

import (
	"errors"
	"strings"
)

const Hyphen = "-"
const Underscore = "_"
const DoubleHyphens = Hyphen + Hyphen
const HyphenUnderscore = Hyphen + Underscore
const UnderscoreHyphen = Underscore + Hyphen

var underscore2hyphens = strings.NewReplacer(Underscore, DoubleHyphens)

var KeyEmptyError = errors.New("Key is empty")

// Because rfc822 http headers disallow _ in names,
// in $meta_name two hyphens in a row (--) will be translated to an underscore(_).
// https://archive.org/developers/ias3.html
//
// If you wanna create a metadata key with two hyphens in a row, You should use Item Metadata API instead.
var KeyDoubleHyphensError = errors.New("Key contains double hyphens (" + DoubleHyphens + ")")

// Although I found through testing that the combination of _- is legal in some cases,
// in order to prevent accidents and more complex processing code, I still regard it as illegal.
//
// If you wanna create a metadata key with -_ or _-, You should use Item Metadata API instead.
var KeyHyphenWithUnderscoreError = errors.New("Key contains -_ or _-, which is not allowed")

// Although we can create an item metadata keys starting with underscore via the S3 API,
// it would lead to an corrupted _meta.xml. (like `<_key>value</_key>`) and IA's /editxml/
// tool would also prevent you from editing the metadata before you fix the key.
var KeyInvalidStartError = errors.New("Key must start with a lowercase a-z letter")

// IA's metadata keys are always lowercase.
// If you try to create a metadata key with uppercase letters, IA will normalize it to lowercase.
var KeyuUpcaseError = errors.New("Key must be lowercase")

// may only contain characters: a-z 0-9 _ - .
var KeyInllegalCharError = errors.New("Key contains illegal characters")

// Key is too long (> 256 characters)
var KeyTooLongError = errors.New("Key is too long")

// TODO: Item Metadata API option
func isValidKey(k string) error {
	if len(k) == 0 {
		return KeyEmptyError
	}
	if len(k) > 256 {
		return KeyTooLongError
	}

	before := k
	k = strings.ToLower(k)
	if before != k {
		return KeyuUpcaseError
	}

	if k[0] < 'a' || k[0] > 'z' {
		return KeyInvalidStartError
	}

	for _, c := range k[1:] {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '_' || c == '-' || c == '.' {
			continue
		}
		return KeyInllegalCharError
	}

	if strings.Contains(k, DoubleHyphens) {
		return KeyDoubleHyphensError
	}
	if strings.Contains(k, HyphenUnderscore) || strings.Contains(k, UnderscoreHyphen) {
		return KeyHyphenWithUnderscoreError
	}

	return nil
}
