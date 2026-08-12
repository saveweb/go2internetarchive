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
	"time"

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
	if err := uploadFile(t.Context(), client, "identifier", path, "artifact", nil, 1, 1, nil); !errors.Is(err, want) {
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
	if err := uploadFile(t.Context(), client, "identifier", localPath, "dir/artifact?#%.warc", nil, 1, 1, nil); err != nil {
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
	if err := uploadFile(ctx, client, "identifier", localPath, "artifact", nil, 1, 1, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("uploadFile error = %v, want context canceled", err)
	}
}

func TestUploadFileTracksProgress(t *testing.T) {
	contents := []byte("artifact contents")
	localPath := filepath.Join(t.TempDir(), "artifact")
	if err := os.WriteFile(localPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if _, err := io.Copy(io.Discard, request.Body); err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}
	tracker := newProgressTracker(int64(len(contents)), 1)
	if err := uploadFile(t.Context(), client, "identifier", localPath, "artifact", nil, 1, 1, tracker); err != nil {
		t.Fatal(err)
	}

	got := tracker.snapshot(0, tracker.startedAt, true)
	if got.BytesUploaded != int64(len(contents)) || got.TotalBytes != int64(len(contents)) {
		t.Fatalf("bytes = %d/%d, want %d/%d", got.BytesUploaded, got.TotalBytes, len(contents), len(contents))
	}
	if got.FilesUploaded != 1 || got.TotalFiles != 1 || got.CurrentFile != "" || !got.Done {
		t.Fatalf("progress = %+v", got)
	}
}

func TestReportProgressDoesNotBlockOnSlowConsumer(t *testing.T) {
	progress := make(chan Progress)
	tracker := newProgressTracker(100, 1)
	stop := reportProgress(progress, tracker, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	stop()
}

func TestReportProgressSendsSnapshots(t *testing.T) {
	progress := make(chan Progress, 1)
	tracker := newProgressTracker(100, 1)
	tracker.startFile("artifact")
	tracker.addBytes(25)
	stop := reportProgress(progress, tracker, 10*time.Millisecond)

	select {
	case got := <-progress:
		if got.BytesUploaded != 25 || got.TotalBytes != 100 || got.CurrentFile != "artifact" || got.Done {
			t.Fatalf("periodic progress = %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for periodic progress")
	}

	tracker.addBytes(75)
	tracker.finishFile()
	stop()
	select {
	case got := <-progress:
		if got.BytesUploaded != 100 || got.FilesUploaded != 1 || !got.Done {
			t.Fatalf("final progress = %+v", got)
		}
	default:
		t.Fatal("final progress was not sent")
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
