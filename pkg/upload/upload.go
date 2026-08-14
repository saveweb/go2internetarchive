package upload

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/saveweb/go2internetarchive/pkg/iaidentifier"
	"github.com/saveweb/go2internetarchive/pkg/iautils"
	"github.com/saveweb/go2internetarchive/pkg/metadata"
	"github.com/saveweb/go2internetarchive/pkg/utils"
	"github.com/schollz/progressbar/v3"
)

var S3Endpoint = "https://s3.us.archive.org/"

// Progress is a snapshot of an upload's progress. BytesUploaded counts bytes
// read by the HTTP transport. Done means that progress reporting has stopped,
// not that the upload succeeded; the Upload return value confirms acceptance.
type Progress struct {
	BytesUploaded  int64
	TotalBytes     int64
	BytesPerSecond int64
	FilesUploaded  int
	TotalFiles     int
	CurrentFile    string
	Elapsed        time.Duration
	Done           bool
}

type progressTracker struct {
	mu            sync.Mutex
	startedAt     time.Time
	bytesUploaded int64
	totalBytes    int64
	filesUploaded int
	totalFiles    int
	currentFile   string
}

func newProgressTracker(totalBytes int64, totalFiles int) *progressTracker {
	return &progressTracker{
		startedAt:  time.Now(),
		totalBytes: totalBytes,
		totalFiles: totalFiles,
	}
}

func (t *progressTracker) addBytes(n int64) {
	t.mu.Lock()
	t.bytesUploaded += n
	t.mu.Unlock()
}

func (t *progressTracker) startFile(remotePath string) {
	t.mu.Lock()
	t.currentFile = remotePath
	t.mu.Unlock()
}

func (t *progressTracker) finishFile() {
	t.mu.Lock()
	t.filesUploaded++
	t.currentFile = ""
	t.mu.Unlock()
}

func (t *progressTracker) snapshot(previousBytes int64, previousAt time.Time, done bool) Progress {
	now := time.Now()
	t.mu.Lock()
	progress := Progress{
		BytesUploaded: t.bytesUploaded,
		TotalBytes:    t.totalBytes,
		FilesUploaded: t.filesUploaded,
		TotalFiles:    t.totalFiles,
		CurrentFile:   t.currentFile,
		Elapsed:       now.Sub(t.startedAt),
		Done:          done,
	}
	t.mu.Unlock()

	if elapsed := now.Sub(previousAt); elapsed > 0 {
		progress.BytesPerSecond = int64(float64(progress.BytesUploaded-previousBytes) / elapsed.Seconds())
	}
	return progress
}

type trackingReader struct {
	reader  io.Reader
	tracker *progressTracker
}

func (r *trackingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.tracker.addBytes(int64(n))
	return n, err
}

func reportProgress(progress chan<- Progress, tracker *progressTracker, interval time.Duration) func() {
	stop := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		previousBytes := int64(0)
		previousAt := tracker.startedAt
		for {
			select {
			case <-ticker.C:
				snapshot := tracker.snapshot(previousBytes, previousAt, false)
				select {
				case progress <- snapshot:
				default:
				}
				previousBytes = snapshot.BytesUploaded
				previousAt = time.Now()
			case <-stop:
				snapshot := tracker.snapshot(previousBytes, previousAt, true)
				select {
				case progress <- snapshot:
				default:
				}
				return
			}
		}
	}()

	return func() {
		close(stop)
		<-stopped
	}
}

func getSize(file string) (int64, error) {
	finfo, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	if finfo.IsDir() {
		return 0, fmt.Errorf("file should not be a directory: %s", file)
	}
	return finfo.Size(), nil
}

func getTotalSize(files map[string]string) (int64, error) {
	var totalSize int64
	for _, localPath := range files {
		size, err := getSize(localPath)
		if err != nil {
			slog.Error("get size failed", "err", err, "localPath", localPath)
			return 0, err
		}
		totalSize += size
	}

	return totalSize, nil
}

func checkRemoteFilenames(files map[string]string) error {
	for remotePath := range files {
		if len(remotePath) == 0 {
			return fmt.Errorf("remotePath should not be empty")
		}
		if strings.HasPrefix(remotePath, "/") {
			// TODO: should we remove the leading /?
			return fmt.Errorf("remotePath should not start with /")
		}
		if strings.HasSuffix(remotePath, "/") {
			return fmt.Errorf("remotePath should not end with /")
		}
	}

	return nil
}

func uploadFile(ctx context.Context, client *http.Client, identifier, localPath, remotePath string, headers map[string]string, current, total int, tracker *progressTracker) error {
	// assert localPath exists
	finfo, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	contentLength := finfo.Size()

	freader, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer freader.Close()

	var body io.Reader = freader
	if tracker != nil {
		tracker.startFile(remotePath)
		body = &trackingReader{reader: body, tracker: tracker}
	} else {
		bar := progressbar.DefaultBytes(contentLength, fmt.Sprintf("[%d/%d] %s", current, total, remotePath))
		progressReader := progressbar.NewReader(freader, bar)
		body = &progressReader
	}

	requestURL, err := buildUploadURL(identifier, remotePath)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", requestURL, body)
	if err != nil {
		return err
	}
	req.ContentLength = contentLength

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("upload failed", "err", err, "url", req.URL)
		return err
	}
	defer resp.Body.Close()

	// slog.Info("req headers", "%v", resp.Request.Header, "req url", resp.Request.URL)
	if resp.StatusCode != http.StatusOK {
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		slog.Info("resp", "body", string(body[:n]))

		slog.Error("resp", "headers", resp.Header)
		return fmt.Errorf("upload failed: %s", resp.Status)
	}
	if tracker != nil {
		tracker.finishFile()
	}
	return nil
}

func buildUploadURL(identifier, remotePath string) (string, error) {
	base, err := url.Parse(S3Endpoint)
	if err != nil {
		return "", fmt.Errorf("parse S3 endpoint: %w", err)
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	base.Path += identifier + "/" + remotePath
	return base.String(), nil
}

// Upload files to Internet Archive
//
//	meta: map[key]values, // key should be in lowercase
//	files: map[remotePath]localPath
func Upload(identifier string, files map[string]string, meta map[string][]string, accKey, secKey string) error {
	return UploadWithProgress(identifier, files, meta, accKey, secKey, nil)
}

// UploadContext uploads files using the supplied context and HTTP client.
func UploadContext(ctx context.Context, client *http.Client, identifier string, files map[string]string, meta map[string][]string, accKey, secKey string) error {
	return UploadContextWithProgress(ctx, client, identifier, files, meta, accKey, secKey, nil)
}

// UploadWithProgress uploads files and sends progress snapshots once per second.
// Sending is non-blocking, including the final snapshot, and the caller remains
// responsible for closing progress.
func UploadWithProgress(identifier string, files map[string]string, meta map[string][]string, accKey, secKey string, progress chan<- Progress) error {
	return UploadContextWithProgress(context.Background(), &http.Client{}, identifier, files, meta, accKey, secKey, progress)
}

// UploadContextWithProgress uploads files using the supplied context and HTTP
// client. It sends progress snapshots once per second and attempts one final
// snapshot. Sending is non-blocking, and the caller remains responsible for
// closing progress.
func UploadContextWithProgress(ctx context.Context, client *http.Client, identifier string, files map[string]string, meta map[string][]string, accKey, secKey string, progress chan<- Progress) error {
	if client == nil {
		return fmt.Errorf("http client is required")
	}
	if err := iaidentifier.IsValidIdentifier(identifier); err != nil {
		return err
	}

	meta["scanner"] = append(meta["scanner"], "saveweb/go2internetarchive "+utils.GetVersion())

	headers, err := metadata.ToS3Headers(meta)
	if err != nil {
		return err
	}

	if err := checkRemoteFilenames(files); err != nil {
		return err
	}

	filesOnline, err := iautils.GetFilesOnlineContext(ctx, client, identifier)
	if err != nil {
		// pass
	} else {
		for fileToUploadTo, localFile := range files {
			for _, fileOnline := range filesOnline {
				if fileToUploadTo == fileOnline.Name {
					localFileSize, err := getSize(localFile)
					if err != nil {
						return err
					}
					localFileSizeStr := fmt.Sprintf("%d", localFileSize)
					if localFileSizeStr == fileOnline.Size {
						// slog.Info("file already exists and is the same, skipping...", "file", fileToUploadTo, "size", localFileSize)
						// delete(files, fileToUploadTo)

						sha1sumLocal, err := utils.SHA1SUM(localFile)
						if err != nil {
							return err
						}
						if sha1sumLocal == fileOnline.SHA1 {
							slog.Info("file already exists and is the same, skipping...", "file", fileToUploadTo, "size", localFileSize, "sha1", sha1sumLocal)
							delete(files, fileToUploadTo)
						} else {
							slog.Warn("file already exists, but sha1 is different", "file", fileToUploadTo, "localSHA1", sha1sumLocal, "onlineSHA1", fileOnline.SHA1)
						}
					} else {
						slog.Warn("file already exists, but size is different", "file", fileToUploadTo, "localSize", localFileSize, "onlineSize", fileOnline.Size)
					}
				}
			}
		}
	}

	TotalSize, err := getTotalSize(files)
	if err != nil {
		return err
	}

	var stopProgress func()
	var tracker *progressTracker
	if progress != nil {
		tracker = newProgressTracker(TotalSize, len(files))
		stopProgress = reportProgress(progress, tracker, time.Second)
		defer stopProgress()
	}

	headers["authorization"] = fmt.Sprintf("LOW %s:%s", accKey, secKey)
	headers["user-agent"] = "saveweb/go2internetarchive"
	headers["x-archive-auto-make-bucket"] = "1"
	headers["x-archive-size-hint"] = fmt.Sprintf("%d", TotalSize)

	headers["x-archive-queue-derive"] = "0" // default to disable derive

	current := 0
	for remotePath, localPath := range files {
		current++
		if current >= len(files) {
			// enable derive for the last file
			headers["x-archive-queue-derive"] = "1"
		}

		err := uploadFile(ctx, client, identifier, localPath, remotePath, headers, current, len(files), tracker)
		if err != nil {
			return err
		}
	}

	return nil
}
