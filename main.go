package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/saveweb/go2internetarchive/pkg/iaidentifier"
	"github.com/saveweb/go2internetarchive/pkg/upload"
	"github.com/saveweb/go2internetarchive/pkg/utils"
	"github.com/tus/tusd/v2/pkg/filelocker"
	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

// configFile mirrors the structure of router.json.
type configFile struct {
	Projects map[string]projectConfig `json:"projects"`
}

// projectConfig mirrors a single project entry in router.json.
// Each metadata value is a {{DATE}}-templated string; the placeholder is
// resolved per-upload via an identifierIssuer.
type projectConfig struct {
	S3KeyPath  string              `json:"s3keypath"`
	Identifier string              `json:"identifier"`
	Metadata   map[string][]string `json:"metadata"`
}

type Router struct {
	Project string

	cfg    projectConfig
	accKey string
	secKey string
}

// identifierIssuer hands out {{DATE}} values (UTC, "20060102150405") for a
// single project, guaranteeing no two values collide.
//
// It is serialized by mu and blocks while the current second has already been
// issued: callers wait until the next whole second before receiving a fresh
// value. This bounds a project to at most one issued date per second, so
// back-to-back uploads never resolve to the same IA item identifier.
type identifierIssuer struct {
	mu      sync.Mutex
	lastSec int64 // unix seconds of the last issued date
}

func (i *identifierIssuer) issue() string {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now().UTC()
	for now.Unix() <= i.lastSec {
		// the current second was already handed out; wait until the next one.
		next := now.Truncate(time.Second).Add(time.Second)
		time.Sleep(time.Until(next))
		now = time.Now().UTC()
	}
	i.lastSec = now.Unix()
	return now.Format("20060102150405")
}

// issuers is a process-wide registry of per-project issuers.
var (
	issuersMu sync.Mutex
	issuers   = map[string]*identifierIssuer{}
)

func issuerFor(project string) *identifierIssuer {
	issuersMu.Lock()
	defer issuersMu.Unlock()
	if iss, ok := issuers[project]; ok {
		return iss
	}
	iss := &identifierIssuer{}
	issuers[project] = iss
	return iss
}

func (u *Router) LoadConfig() error {
	log.Printf("Loading config for project %s", u.Project)

	raw, err := os.ReadFile("./router.json")
	if err != nil {
		return fmt.Errorf("read router.json: %w", err)
	}

	var cfg configFile
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse router.json: %w", err)
	}

	pc, ok := cfg.Projects[u.Project]
	if !ok {
		return fmt.Errorf("project %q not found in router.json", u.Project)
	}
	u.cfg = pc

	u.accKey, u.secKey, err = utils.ReadKeysFromFile(pc.S3KeyPath)
	if err != nil {
		return fmt.Errorf("read s3 keys from %s: %w", pc.S3KeyPath, err)
	}

	return nil
}

// resolveDate replaces {{DATE}} in a template with the given date.
func resolveDate(tmpl, date string) string {
	return strings.ReplaceAll(tmpl, "{{DATE}}", date)
}

func (u *Router) Upload(localFilepath, remotePath string) error {
	// Each upload is its own IA item, so {{DATE}} is resolved independently
	// here. The issuer guarantees the value is unique within this project.
	date := issuerFor(u.Project).issue()

	identifier := resolveDate(u.cfg.Identifier, date)
	if err := iaidentifier.IsValidIdentifier(identifier); err != nil {
		return fmt.Errorf("invalid identifier %q: %w", identifier, err)
	}

	log.Printf("Uploading to IA item %s (date=%s, file=%s)", identifier, date, remotePath)

	// Copy metadata and resolve {{DATE}} with the same date, so the identifier
	// and metadata stay consistent. The copy also avoids mutating u.cfg across
	// uploads (upload.Upload appends to the scanner field).
	meta := make(map[string][]string, len(u.cfg.Metadata))
	for k, vs := range u.cfg.Metadata {
		resolved := make([]string, len(vs))
		for i, v := range vs {
			resolved[i] = resolveDate(v, date)
		}
		meta[k] = resolved
	}

	files := map[string]string{
		remotePath: localFilepath,
	}

	return upload.Upload(identifier, files, meta, u.accKey, u.secKey)
}

func handleUploadEvent(event tusd.HookEvent) {
	upload := event.Upload
	log.Printf("Upload %s finished, size=%d\n", upload.ID, upload.Size)
	log.Printf("  metadata: %v", upload.MetaData)

	project, _ := upload.MetaData["project"]
	filename, _ := upload.MetaData["filename"]

	if storage := upload.Storage; storage != nil {
		switch storage["Type"] {
		case "filestore":
			storagePath := storage["Path"]
			log.Printf("  file path: %s", storagePath)
			log.Printf("  info path: %s", storage["InfoPath"])

			uploader := Router{
				Project: project,
			}
			if err := uploader.LoadConfig(); err != nil {
				slog.Error("load config failed", "project", project, "err", err)
				return
			}

			if err := uploader.Upload(storagePath, filename); err != nil {
				slog.Error("upload failed", "project", project, "file", storagePath, "err", err)
			}

		case "s3store", "gcsstore":
			slog.Error("unsupported storage type", "type", storage["Type"])
			log.Printf("  bucket: %s", storage["Bucket"])
			log.Printf("  key:    %s", storage["Key"])
		}
	}
}

func handleRouterJSON(w http.ResponseWriter, r *http.Request) {
	router := Router{}
	if err := router.LoadConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(router.cfg)
}

func main() {
	if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
		if err := os.Mkdir("./uploads", 0755); err != nil {
			log.Fatal("failed to create uploads directory: %w", err)
		}
	}
	// check permissions
	if err := os.Chmod("./uploads", 0755); err != nil {
		log.Fatal("failed to set permissions for uploads directory: %w", err)
	}
	store := filestore.New("./uploads")
	locker := filelocker.New("./uploads")

	composer := tusd.NewStoreComposer()
	store.UseIn(composer)
	locker.UseIn(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:              "/files/",
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
	})
	if err != nil {
		log.Fatalf("unable to create handler: %s", err)
	}

	go func() {
		for {
			event := <-handler.CompleteUploads
			go handleUploadEvent(event)
		}
	}()
	// Right now, nothing has happened since we need to start the HTTP server on
	// our own. In the end, tusd will start listening on and accept request at
	// http://localhost:8080/files
	http.Handle("/files/", http.StripPrefix("/files/", handler))
	http.Handle("/files", http.StripPrefix("/files", handler))
	http.Handle("/router.json", http.HandlerFunc(handleRouterJSON))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("unable to listen: %s", err)
	}
}
