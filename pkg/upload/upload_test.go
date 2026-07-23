package upload

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saveweb/go2internetarchive/pkg/utils"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestUploadFileReturnsNetworkError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact")
	if err := os.WriteFile(path, []byte("artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := errors.New("network unavailable")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, want
	})}
	if err := uploadFile(t.Context(), client, "identifier", path, "artifact", nil, 1, 1); !errors.Is(err, want) {
		t.Fatalf("uploadFile error = %v, want %v", err, want)
	}
}

func TestUploadFileEscapesRemotePath(t *testing.T) {
	localPath := filepath.Join(t.TempDir(), "artifact")
	if err := os.WriteFile(localPath, []byte("artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got *http.Request
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		got = request
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}
	if err := uploadFile(t.Context(), client, "identifier", localPath, "dir/artifact?#%.warc", nil, 1, 1); err != nil {
		t.Fatal(err)
	}
	if got.URL.EscapedPath() != "/identifier/dir/artifact%3F%23%25.warc" || got.URL.RawQuery != "" || got.URL.Fragment != "" {
		t.Fatalf("upload URL = %q", got.URL.String())
	}
}

func TestUploadFileHonorsContext(t *testing.T) {
	localPath := filepath.Join(t.TempDir(), "artifact")
	if err := os.WriteFile(localPath, []byte("artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return nil, request.Context().Err()
	})}
	if err := uploadFile(ctx, client, "identifier", localPath, "artifact", nil, 1, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("uploadFile error = %v, want context canceled", err)
	}
}

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
