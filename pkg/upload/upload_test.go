package upload

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saveweb/go2internetarchive/pkg/utils"
)

func Test_Upload_Fail(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Test case
	testCases := []struct {
		name       string
		meta       map[string][]string
		files      map[string]string
		wantPrefix string
	}{
		{
			name: "scanner field missing test",
			meta: map[string][]string{
				"title": {"Test Item"},
			},
			files: map[string]string{
				"test.txt": tmpFile,
			},
			wantPrefix: "saveweb/go2internetarchive ",
		},
		{
			name: "scanner field exist test",
			meta: map[string][]string{
				"title":   {"Test Item"},
				"scanner": {"myarchiver"},
			},
			files: map[string]string{
				"test.txt": tmpFile,
			},
			wantPrefix: "saveweb/go2internetarchive ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call Upload
			err := Upload("test_identifier", tc.files, tc.meta, "test_access", "test_secret")
			if err == nil {
				// We expect an error due to invalid credentials, but we can still check if scanner was appended
				t.Fatal("expected error due to invalid credentials")
			}

			// Check if scanner field was properly appended
			scanners := tc.meta["scanner"]
			if len(scanners) == 0 {
				t.Fatal("scanner field not added")
			}

			lastScanner := scanners[len(scanners)-1]
			if !strings.HasPrefix(lastScanner, tc.wantPrefix) {
				t.Errorf("scanner field doesn't have correct prefix\ngot: %v\nwant prefix: %v",
					lastScanner, tc.wantPrefix)
			}

			// Verify version is included
			version := utils.GetVersion()
			expectedScanner := tc.wantPrefix + version
			if lastScanner != expectedScanner {
				t.Errorf("unexpected scanner value\ngot: %v\nwant: %v",
					lastScanner, expectedScanner)
			}
		})
	}
}
