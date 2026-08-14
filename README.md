# GO2INTERNETARCHIVE - Yet another IA S3 Client in Go (WIP)

> This is a work in progress project. The packages and functions may change at any time.

# Usage

```go
identifier := "some_random_identifier"
files := map[string]string{
	"filepath/on/ia": "path/to/your/local/file",
	"file2":          "file2",
}

meta := map[string][]string{
	"title":       {"test - dsasdasdasadsa"},
	"collection":  {"test_collection"},
	"creator":     {"author1", "author2", "someone3"}, // multiple values
	"description": {"<body>hello world</body>"}, // description, plain text or html
	"mediatype":   {"image"},
	"mymeta1":       {"hello This+is+mymeta1, !@#$%^&*()_+{}|:\"<>? 你好👋"},
}

// the first line is the access key, and the second line is the secret key.
acckey, seckey, err := utils.ReadKeysFromFile("path/to/your/keys.txt")
if err != nil {
	panic(err)
}

err := upload.Upload(identifier, files, meta, acckey, seckey)
if err != nil {
	panic(err)
}
```

Files are uploaded sequentially from smallest to largest. Files with the same
size are ordered by their remote path.

To receive upload statistics, pass a channel to
`UploadWithProgress`. The uploader sends a snapshot once per second and a final snapshot when it returns. Sends are non-blocking, so a slow consumer may miss snapshots, including the final one.

The caller owns and closes the channel.

```go
progress := make(chan upload.Progress, 1)
go func() {
	for snapshot := range progress {
		log.Printf("uploaded %d/%d bytes (%d bytes/s)",
			snapshot.BytesUploaded, snapshot.TotalBytes, snapshot.BytesPerSecond)
	}
}()

err = upload.UploadWithProgress(identifier, files, meta, acckey, seckey, progress)
close(progress)
if err != nil {
	panic(err)
}
```

Realworld example -> <https://github.com/saveweb/aixifan/blob/main/pkg/uploader/up.go>


## Metadata

### Metadata Key

- [x] Handle all kinds of illegal characters and edge cases!

### Metadata Value

- [x] Replace XML illegal characters with U+FFFD (�)

## Uploading

- [x] PUT
   - [ ] Queue derive
      - [x] if `true`: trigger derive for the last PUT
      - [ ] if `false`: do nothing
   - [ ] Checksum
      - [ ] retry if the checksum is different
      - [ ] skip upload if the checksum is the same
   - [ ] Retries
   - [x] `x-archive-size-hint`
- [ ] Multipart Upload
   - [ ] Resumable Upload
- [ ] DELETE
- [x] Implement the `upload` function.

## Misc

- [ ] ini parser (`.config/internetarchive/ia.ini`)
- [ ] Multi Account Support
- [ ] Download
- [ ] Upload
- [ ] Metadata
- [ ] Search
- [ ] Task
