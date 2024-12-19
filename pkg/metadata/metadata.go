package metadata

import (
	"errors"
	"fmt"
	"log/slog"
)

var ErrEmptyValue = errors.New("empty value")
var ErrNLessThanOne = errors.New("n must be greater than 0")

func toS3HeaderKey(k string, n int) (string, error) {
	if err := isValidKey(k); err != nil {
		return "", err
	}
	if n < 1 {
		return "", ErrNLessThanOne
	}

	// f'x-archive-{meta_type}{i:02d}-{meta_key}'
	return fmt.Sprintf("x-archive-meta%02d-%s", n, k), nil
}

func toS3HeaderValue(v string) string {
	r, replaced := ReplaceIllegalXMLChars(v)
	if replaced {
		slog.Warn("Illegal XML characters were replaced.")
	}
	iaS3Escaped := uriEscape(r)
	return iaS3Escaped
}

func ToS3Headers(m map[string][]string) (map[string]string, error) {
	headers := make(map[string]string)

	for k, v := range m {
		if len(v) == 0 {
			return nil, ErrEmptyValue
		}

		for i, vv := range v {
			n := i + 1
			s3key, err := toS3HeaderKey(k, n)
			if err != nil {
				return nil, err
			}

			if _, ok := headers[s3key]; ok {
				panic("duplicate key, this should never happen")
			}

			s3value := toS3HeaderValue(vv)
			// TODO: what will happen if the value is empty?
			headers[s3key] = s3value
		}
	}

	return headers, nil
}
