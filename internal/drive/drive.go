package drive

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

// DriveSync holds the authenticated Drive service and context needed for sync.
type DriveSync struct {
	service   *gdrive.Service
	identity  *age.HybridIdentity
	notesDir  string
	configDir string
}

type driveFileEntry struct {
	ID      string
	Name    string
	RelPath string
}

// Setup runs an interactive OAuth2 setup flow: reads a credentials JSON file,
// opens the browser for sign-in, catches the callback on a local server,
// and stores the encrypted tokens.
func Setup(configDir string, identity *age.HybridIdentity) error {
	fmt.Println("Google Drive Setup")
	fmt.Println("==================")
	fmt.Println("To use Drive sync, you need a Google Cloud project with the Drive API enabled.")
	fmt.Println("1. Go to https://console.cloud.google.com/apis/credentials")
	fmt.Println("2. Create an OAuth 2.0 Client ID (Desktop application)")
	fmt.Println("3. Download the JSON credentials file")
	fmt.Print("\nPath to credentials JSON: ")

	reader := bufio.NewReader(os.Stdin)
	credPath, _ := reader.ReadString('\n')
	credPath = strings.TrimSpace(credPath)

	credData, err := os.ReadFile(credPath)
	if err != nil {
		return fmt.Errorf("read credentials: %w", err)
	}

	oauthConfig, err := google.ConfigFromJSON(credData, gdrive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}

	token, err := oauthViaLocalServer(oauthConfig)
	if err != nil {
		return fmt.Errorf("oauth flow: %w", err)
	}

	tokenData, err := json.Marshal(token)
	if err != nil {
		return err
	}

	encPath := filepath.Join(configDir, "google-credentials.json.age")
	if err := crypto.EncryptToFile(encPath, tokenData, identity.Recipient()); err != nil {
		return fmt.Errorf("save encrypted credentials: %w", err)
	}

	credEncPath := filepath.Join(configDir, "google-client.json.age")
	if err := crypto.EncryptToFile(credEncPath, credData, identity.Recipient()); err != nil {
		return fmt.Errorf("save encrypted client config: %w", err)
	}

	fmt.Println("\nDrive setup complete! Credentials stored encrypted.")
	return nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func oauthViaLocalServer(cfg *oauth2.Config) (*oauth2.Token, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	cfg.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("OAuth state mismatch (possible CSRF)")
			fmt.Fprint(w, "<html><body><h2>Authentication failed.</h2><p>State mismatch.</p></body></html>")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback: %s", r.URL.Query().Get("error"))
			fmt.Fprint(w, "<html><body><h2>Authentication failed.</h2><p>You can close this tab.</p></body></html>")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "<html><body><h2>Authentication successful!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Close()

	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Printf("\nOpening browser for Google sign-in...\n")
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically.\nOpen this URL manually:\n%s\n", authURL)
	}
	fmt.Println("Waiting for authentication...")

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("OAuth authentication timed out after 5 minutes")
	}

	return cfg.Exchange(context.Background(), code)
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

// Sync walks the notes directory and syncs files to Google Drive.
func Sync(notesDir, configDir string, identity *age.HybridIdentity) error {
	ds, err := newDriveSync(notesDir, configDir, identity)
	if err != nil {
		return err
	}
	return ds.syncAll()
}

type persistingTokenSource struct {
	base      oauth2.TokenSource
	mu        sync.Mutex
	lastToken *oauth2.Token
	configDir string
	identity  *age.HybridIdentity
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.base.Token()
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.lastToken == nil || tok.AccessToken != p.lastToken.AccessToken {
		p.lastToken = tok
		data, err := json.Marshal(tok)
		if err == nil {
			encPath := filepath.Join(p.configDir, "google-credentials.json.age")
			if encErr := crypto.EncryptToFile(encPath, data, p.identity.Recipient()); encErr != nil {
				log.Printf("drive: failed to persist refreshed token: %v", encErr)
			}
		}
	}

	return tok, nil
}

func newDriveSync(notesDir, configDir string, identity *age.HybridIdentity) (*DriveSync, error) {
	credData, err := crypto.DecryptFile(filepath.Join(configDir, "google-client.json.age"), identity)
	if err != nil {
		return nil, fmt.Errorf("load client config: %w (run 'pq-notes drive setup' first)", err)
	}

	oauthConfig, err := google.ConfigFromJSON(credData, gdrive.DriveFileScope)
	if err != nil {
		return nil, err
	}

	tokenData, err := crypto.DecryptFile(filepath.Join(configDir, "google-credentials.json.age"), identity)
	if err != nil {
		return nil, fmt.Errorf("load token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(tokenData, &token); err != nil {
		return nil, err
	}

	ctx := context.Background()
	pts := &persistingTokenSource{
		base:      oauthConfig.TokenSource(ctx, &token),
		configDir: configDir,
		identity:  identity,
	}
	client := oauth2.NewClient(ctx, pts)

	svc, err := gdrive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &DriveSync{
		service:   svc,
		identity:  identity,
		notesDir:  notesDir,
		configDir: configDir,
	}, nil
}

func escapeDriveQuery(s string) string {
	return strings.ReplaceAll(s, `'`, `\'`)
}

const rootFolderName = "pq-notes"

func (ds *DriveSync) syncAll() error {
	fmt.Println("Syncing notes to Google Drive...")

	rootID, err := ds.findOrCreateFolder(rootFolderName, "root")
	if err != nil {
		return fmt.Errorf("create root folder: %w", err)
	}

	folderCache := map[string]string{"": rootID}
	var count int

	err = filepath.Walk(ds.notesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".pq-notes" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".age" {
			return nil
		}

		relPath, err := filepath.Rel(ds.notesDir, path)
		if err != nil {
			return fmt.Errorf("relative path for %s: %w", path, err)
		}
		parentDir := filepath.Dir(relPath)

		parentID, err := ds.ensureFolderPath(parentDir, rootID, folderCache)
		if err != nil {
			return fmt.Errorf("create folder %s: %w", parentDir, err)
		}

		_, err = ds.uploadOrUpdateFile(path, filepath.Base(relPath), parentID)
		if err != nil {
			fmt.Printf("  Error uploading %s: %v\n", relPath, err)
			return nil
		}

		count++
		fmt.Printf("  Synced: %s\n", relPath)
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Sync complete! %d files synced.\n", count)
	return nil
}

func (ds *DriveSync) ensureFolderPath(relDir, rootID string, cache map[string]string) (string, error) {
	if relDir == "." || relDir == "" {
		return rootID, nil
	}

	if id, ok := cache[relDir]; ok {
		return id, nil
	}

	parts := strings.Split(relDir, string(filepath.Separator))
	parentID := rootID
	built := ""

	for _, part := range parts {
		if built == "" {
			built = part
		} else {
			built = built + string(filepath.Separator) + part
		}

		if id, ok := cache[built]; ok {
			parentID = id
			continue
		}

		id, err := ds.findOrCreateFolder(part, parentID)
		if err != nil {
			return "", err
		}
		cache[built] = id
		parentID = id
	}

	return parentID, nil
}

func (ds *DriveSync) findOrCreateFolder(name, parentID string) (string, error) {
	q := fmt.Sprintf("name='%s' and '%s' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false",
		escapeDriveQuery(name), escapeDriveQuery(parentID))
	list, err := ds.service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return "", err
	}

	if len(list.Files) > 0 {
		return list.Files[0].Id, nil
	}

	folder := &gdrive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentID},
	}
	created, err := ds.service.Files.Create(folder).Fields("id").Do()
	if err != nil {
		return "", err
	}
	return created.Id, nil
}

func (ds *DriveSync) uploadOrUpdateFile(localPath, name, parentID string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	q := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false",
		escapeDriveQuery(name), escapeDriveQuery(parentID))
	list, err := ds.service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return "", err
	}

	if len(list.Files) > 0 {
		id := list.Files[0].Id
		_, err = ds.service.Files.Update(id, &gdrive.File{}).Media(f).Do()
		return id, err
	}

	meta := &gdrive.File{
		Name:    name,
		Parents: []string{parentID},
	}
	created, err := ds.service.Files.Create(meta).Media(f).Fields("id").Do()
	if err != nil {
		return "", err
	}
	return created.Id, nil
}

func (ds *DriveSync) deleteOrphans(rootID string, keepIDs map[string]bool) (int, error) {
	allFileIDs, err := ds.listAllFileIDs(rootID)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, id := range allFileIDs {
		if keepIDs[id] {
			continue
		}
		if err := ds.service.Files.Delete(id).Do(); err != nil {
			log.Printf("drive: failed to delete orphan %s: %v", id, err)
			continue
		}
		deleted++
	}
	return deleted, nil
}

func (ds *DriveSync) listAllFileIDs(folderID string) ([]string, error) {
	var fileIDs []string
	var folderIDs []string

	q := fmt.Sprintf("'%s' in parents and trashed=false", escapeDriveQuery(folderID))
	pageToken := ""
	for {
		call := ds.service.Files.List().Q(q).Fields("nextPageToken, files(id, mimeType)")
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		list, err := call.Do()
		if err != nil {
			return nil, err
		}

		for _, f := range list.Files {
			if f.MimeType == "application/vnd.google-apps.folder" {
				folderIDs = append(folderIDs, f.Id)
			} else {
				fileIDs = append(fileIDs, f.Id)
			}
		}

		if list.NextPageToken == "" {
			break
		}
		pageToken = list.NextPageToken
	}

	for _, subID := range folderIDs {
		subFiles, err := ds.listAllFileIDs(subID)
		if err != nil {
			return nil, err
		}
		fileIDs = append(fileIDs, subFiles...)
	}

	return fileIDs, nil
}

func (ds *DriveSync) walkDriveTree(folderID, prefix string) ([]driveFileEntry, error) {
	var entries []driveFileEntry

	q := fmt.Sprintf("'%s' in parents and trashed=false", escapeDriveQuery(folderID))
	pageToken := ""
	for {
		call := ds.service.Files.List().Q(q).Fields("nextPageToken, files(id, name, mimeType)")
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		list, err := call.Do()
		if err != nil {
			return nil, err
		}

		for _, f := range list.Files {
			if f.MimeType == "application/vnd.google-apps.folder" {
				// Recurse into folder
				var newPrefix string
				if prefix == "" {
					newPrefix = f.Name
				} else {
					newPrefix = prefix + "/" + f.Name
				}
				subEntries, err := ds.walkDriveTree(f.Id, newPrefix)
				if err != nil {
					return nil, err
				}
				entries = append(entries, subEntries...)
			} else {
				// Add file entry
				var relPath string
				if prefix == "" {
					relPath = f.Name
				} else {
					relPath = prefix + "/" + f.Name
				}
				entries = append(entries, driveFileEntry{
					ID:      f.Id,
					Name:    f.Name,
					RelPath: relPath,
				})
			}
		}

		if list.NextPageToken == "" {
			break
		}
		pageToken = list.NextPageToken
	}

	return entries, nil
}

func (ds *DriveSync) findFolder(name, parentID string) (string, error) {
	q := fmt.Sprintf("name='%s' and '%s' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false",
		escapeDriveQuery(name), escapeDriveQuery(parentID))
	list, err := ds.service.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return "", err
	}
	if len(list.Files) == 0 {
		return "", fmt.Errorf("folder %q not found on Drive", name)
	}
	return list.Files[0].Id, nil
}

// Pull downloads all files from the Drive pq-notes folder to the local notes directory.
// Files that already exist locally are skipped.
func Pull(notesDir, configDir string, identity *age.HybridIdentity) error {
	ds, err := newDriveSync(notesDir, configDir, identity)
	if err != nil {
		return err
	}

	rootID, err := ds.findFolder(rootFolderName, "root")
	if err != nil {
		return fmt.Errorf("no pq-notes folder found on Drive — nothing to pull")
	}

	entries, err := ds.walkDriveTree(rootID, "")
	if err != nil {
		return fmt.Errorf("list Drive files: %w", err)
	}

	var restored int
	for _, entry := range entries {
		localPath := filepath.Join(notesDir, filepath.FromSlash(entry.RelPath))

		if _, err := os.Stat(localPath); err == nil {
			fmt.Printf("  Skipped: %s (already exists)\n", entry.RelPath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(localPath), 0700); err != nil {
			return fmt.Errorf("create directory for %s: %w", entry.RelPath, err)
		}

		resp, err := ds.service.Files.Get(entry.ID).Download()
		if err != nil {
			fmt.Printf("  Error downloading %s: %v\n", entry.RelPath, err)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("  Error reading %s: %v\n", entry.RelPath, err)
			continue
		}

		if err := os.WriteFile(localPath, data, 0600); err != nil {
			return fmt.Errorf("write %s: %w", entry.RelPath, err)
		}

		restored++
		fmt.Printf("  Restored: %s\n", entry.RelPath)
	}

	fmt.Printf("Pull complete! %d files restored.\n", restored)
	return nil
}

// Clean removes files from Drive that no longer exist locally.
func Clean(notesDir, configDir string, identity *age.HybridIdentity) error {
	ds, err := newDriveSync(notesDir, configDir, identity)
	if err != nil {
		return err
	}

	rootID, err := ds.findFolder(rootFolderName, "root")
	if err != nil {
		return fmt.Errorf("no pq-notes folder found on Drive — nothing to clean")
	}

	localFiles := make(map[string]bool)
	err = filepath.Walk(notesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".pq-notes" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".age" {
			return nil
		}
		relPath, err := filepath.Rel(notesDir, path)
		if err != nil {
			return nil
		}
		localFiles[filepath.ToSlash(relPath)] = true
		return nil
	})
	if err != nil {
		return err
	}

	entries, err := ds.walkDriveTree(rootID, "")
	if err != nil {
		return fmt.Errorf("list Drive files: %w", err)
	}

	var deleted int
	for _, entry := range entries {
		if localFiles[entry.RelPath] {
			continue
		}
		if err := ds.service.Files.Delete(entry.ID).Do(); err != nil {
			log.Printf("drive: failed to delete %s: %v", entry.RelPath, err)
			continue
		}
		deleted++
		fmt.Printf("  Deleted from Drive: %s\n", entry.RelPath)
	}

	fmt.Printf("Clean complete! %d files removed from Drive.\n", deleted)
	return nil
}

// SyncFile uploads or updates a single file to Drive.
// Logs errors instead of returning them so it doesn't block the TUI.
func SyncFile(filePath, notesDir, configDir string, identity *age.HybridIdentity) {
	ds, err := newDriveSync(notesDir, configDir, identity)
	if err != nil {
		log.Printf("drive: sync file: %v", err)
		return
	}

	relPath, err := filepath.Rel(notesDir, filePath)
	if err != nil {
		log.Printf("drive: relative path for %s: %v", filePath, err)
		return
	}

	rootID, err := ds.findOrCreateFolder(rootFolderName, "root")
	if err != nil {
		log.Printf("drive: find root folder: %v", err)
		return
	}

	parentDir := filepath.Dir(relPath)
	folderCache := map[string]string{"": rootID}
	parentID, err := ds.ensureFolderPath(parentDir, rootID, folderCache)
	if err != nil {
		log.Printf("drive: ensure folder %s: %v", parentDir, err)
		return
	}

	_, err = ds.uploadOrUpdateFile(filePath, filepath.Base(relPath), parentID)
	if err != nil {
		log.Printf("drive: upload %s: %v", relPath, err)
		return
	}

	log.Printf("drive: synced %s", relPath)
}

// DeleteFile removes a single file from Drive by its relative path.
// Logs errors instead of returning them so it doesn't block the TUI.
func DeleteFile(relPath, notesDir, configDir string, identity *age.HybridIdentity) {
	ds, err := newDriveSync(notesDir, configDir, identity)
	if err != nil {
		log.Printf("drive: delete file: %v", err)
		return
	}

	rootID, err := ds.findFolder(rootFolderName, "root")
	if err != nil {
		log.Printf("drive: find root folder: %v", err)
		return
	}

	entries, err := ds.walkDriveTree(rootID, "")
	if err != nil {
		log.Printf("drive: walk tree: %v", err)
		return
	}

	target := filepath.ToSlash(relPath)
	for _, entry := range entries {
		if entry.RelPath == target {
			if err := ds.service.Files.Delete(entry.ID).Do(); err != nil {
				log.Printf("drive: delete %s: %v", relPath, err)
				return
			}
			log.Printf("drive: deleted %s from Drive", relPath)
			return
		}
	}

	log.Printf("drive: file %s not found on Drive", relPath)
}
