package utils

import (
	"fmt"
	"os"
)

// ReadKeysFromFile reads access key and secret key from a file.
//
// The file should contain at least two lines,
// the first line is the access key, and the second line is the secret key.
func ReadKeysFromFile(file string) (accKey, secKey string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	keys := make([]string, 2)
	_, err = fmt.Fscanf(f, "%s\n%s\n", &keys[0], &keys[1])
	if err != nil {
		return "", "", err
	}

	return keys[0], keys[1], nil
}
