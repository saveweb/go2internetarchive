package upload

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/saveweb/go2internetarchive/pkg/iaidentifier"
	"github.com/saveweb/go2internetarchive/pkg/metadata"
	"github.com/saveweb/go2internetarchive/pkg/utils"
	"github.com/schollz/progressbar/v3"
)

var S3Endpoint = "https://s3.us.archive.org/"

func getTotalSize(files map[string]string) (int64, error) {
	var totalSize int64
	for _, localPath := range files {
		finfo, err := os.Stat(localPath)
		if err != nil {
			return 0, err
		}
		if finfo.IsDir() {
			return 0, fmt.Errorf("localPath should not be a directory: %s", localPath)
		}
		totalSize += finfo.Size()
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

func uploadFile(client *http.Client, identifier, localPath, remotePath string, headers map[string]string, current, total int) error {
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

	bar := progressbar.DefaultBytes(contentLength, fmt.Sprintf("[%d/%d] %s", current, total, remotePath))
	progressReader := progressbar.NewReader(freader, bar)

	req, err := http.NewRequest("PUT", S3Endpoint+identifier+"/"+remotePath, &progressReader)
	if err != nil {
		return err
	}
	req.ContentLength = contentLength

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("upload failed", "err", err, "url", resp.Request.URL)
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
	return nil
}

// Upload files to Internet Archive
//
//	meta: map[key]values, // key should be in lowercase
//	files: map[remotePath]localPath
func Upload(identifier string, files map[string]string, meta map[string][]string, accKey, secKey string) error {
	if err := iaidentifier.IsValidIdentifier(identifier); err != nil {
		return err
	}

	meta["scanner"] = append(meta["scanner"], "saveweb/go2internetarchive "+utils.GetVersion())

	headers, err := metadata.ToS3Headers(meta)
	if err != nil {
		return err
	}

	TotalSize, err := getTotalSize(files)
	if err != nil {
		return err
	}

	if err := checkRemoteFilenames(files); err != nil {
		return err
	}

	client := &http.Client{}

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

		err := uploadFile(client, identifier, localPath, remotePath, headers, current, len(files))
		if err != nil {
			return err
		}
	}

	return nil
}
