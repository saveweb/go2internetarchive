package upload

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/saveweb/go2internetarchive/pkg/iaidentifier"
	"github.com/saveweb/go2internetarchive/pkg/metadata"
	"github.com/schollz/progressbar/v3"
)

var S3Endpoint = "https://s3.us.archive.org/"

// Upload files to Internet Archive
//
//	meta: map[key]values, // key should be in lowercase
//	files: map[remotePath]localPath
func Upload(identifier string, files map[string]string, meta map[string][]string, accKey, secKey string) error {
	if err := iaidentifier.IsValidIdentifier(identifier); err != nil {
		return err
	}

	headers, err := metadata.ToS3Headers(meta)
	if err != nil {
		return err
	}

	client := &http.Client{}

	headers["authorization"] = fmt.Sprintf("LOW %s:%s", accKey, secKey)
	headers["user-agent"] = "saveweb/go2internetarchive"
	headers["x-archive-auto-make-bucket"] = "1"

	headers["x-archive-queue-derive"] = "0" // default to disable derive

	progress := 0
	for remotePath, localPath := range files {
		progress++
		if progress >= len(files) {
			// enable derive for the last file
			headers["x-archive-queue-derive"] = "1"
		}

		if strings.HasSuffix(remotePath, "/") {
			return fmt.Errorf("remotePath should not end with /")
		}

		// assert localPath exists
		finfo, err := os.Stat(localPath)
		if err != nil {
			return err
		}
		if finfo.IsDir() {
			return fmt.Errorf("localPath should not be a directory")
		}

		contentLength := finfo.Size()

		// TODO: size hint
		// headers["x-archive-size-hint"]

		freader, err := os.Open(localPath)
		if err != nil {
			return err
		}
		defer freader.Close()

		bar := progressbar.DefaultBytes(contentLength, fmt.Sprintf("uploading -> %s", remotePath))
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

		if resp.StatusCode != http.StatusOK {
			// slog.Info("req headers", "%v", resp.Request.Header, "req url", resp.Request.URL)
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			slog.Info("resp", "body", string(body[:n]))

			slog.Error("resp", "headers", resp.Header)
			return fmt.Errorf("upload failed: %s", resp.Status)
		}
		fmt.Printf("upload %s to %s/%s\n", localPath, identifier, remotePath)
	}

	return nil
}
