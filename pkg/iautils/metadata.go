package iautils

import (
	"context"
	"encoding/json"
	"net/http"
)

type File struct {
	Name string `json:"name"`
	Size string `json:"size"`
	MD5  string `json:"md5"`
	SHA1 string `json:"sha1"`
}

type MetadataOnline struct {
	Files []File `json:"files"`
}

func getMetadataOnline(identifier string) (MetadataOnline, error) {
	return getMetadataOnlineContext(context.Background(), http.DefaultClient, identifier)
}

func getMetadataOnlineContext(ctx context.Context, client *http.Client, identifier string) (MetadataOnline, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://archive.org/metadata/"+identifier, nil)
	if err != nil {
		return MetadataOnline{}, err
	}
	req.Header.Set("User-Agent", "saveweb/go2internetarchive")
	res, err := client.Do(req)
	if err != nil {
		return MetadataOnline{}, err
	}
	defer res.Body.Close()

	var metadata MetadataOnline
	if err := json.NewDecoder(res.Body).Decode(&metadata); err != nil {
		return MetadataOnline{}, err
	}

	return metadata, nil
}

func GetFilesOnline(identifier string) ([]File, error) {
	return GetFilesOnlineContext(context.Background(), http.DefaultClient, identifier)
}

func GetFilesOnlineContext(ctx context.Context, client *http.Client, identifier string) ([]File, error) {
	metadata, err := getMetadataOnlineContext(ctx, client, identifier)
	if err != nil {
		return nil, err
	}

	return metadata.Files, nil
}
