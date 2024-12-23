# GO2INTERNETARCHIVE - Yet another implementation of the IA S3-like API Client in Go (WIP)

# Usage

```go
	identifier := "some_random_identifier"
	files := map[string]string{
		"filepath/on/ia":      "path/to/your/local/file",
		"file2": "file2",
	}

	meta := map[string][]string{
		"title":       {"test - metadata"},
		"collection":  {"test_collection"},
		"creator":     {"author1", "author2", "someone3"}, // multiple values
		"description": {"<body>hello world</body>"}, // description, plain text or html
		"mediatype":   {"image"},
		"scanner":     {"saveweb"},
		"meta1":       {"hello This+is+meta1, !@#$%^&*()_+{}|:\"<>? 你好👋"},
	}
	acckey := "accessKey"
	seckey := "secretKey"

	err := upload.Upload(identifier, files, meta, acckey, seckey)
	if err != nil {
		panic(err)
	}
```


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

- [ ] ini parser (`.config/internerarchive/ia.ini`)
- [ ] Multi Account Support
- [ ] Download
- [ ] Upload
- [ ] Metadata
- [ ] Search
- [ ] Task