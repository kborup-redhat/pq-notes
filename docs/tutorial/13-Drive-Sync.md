---
title: "Chapter 13: Google Drive Sync"
order: 13
---

# Chapter 13: Google Drive Sync

Local notes are great until your laptop's hard drive dies, or you want to pick up where you left off on a different machine. **Google Drive Sync** solves both problems by mirroring your encrypted `.age` files to the cloud. The files remain encrypted -- Google never sees your plaintext -- so you get the convenience of cloud backup with the security of local-only decryption.

Think of Drive Sync as a safety deposit box at a bank. You put sealed envelopes (encrypted files) into the box. The bank stores them securely, but only you have the key to open the envelopes. If your house burns down, the sealed envelopes are still safe at the bank.

## How It Works

The sync system has five parts:

1. **OAuth2 Setup** -- a one-time interactive flow that authenticates with Google and stores encrypted tokens.
2. **persistingTokenSource** -- a wrapper that automatically saves refreshed tokens so the user never re-authenticates.
3. **Sync (push)** -- walks local `.age` files, mirrors the folder structure on Drive, uploads new or changed files.
4. **Pull** -- downloads files from Drive that don't exist locally (for restoring on a new machine).
5. **Clean** -- deletes files from Drive that no longer exist locally.

Additionally, `SyncFile` and `DeleteFile` provide single-file operations that the TUI calls in fire-and-forget mode when notes are created, edited, or deleted.

## Code Deep Dive

### The DriveSync Struct

All Drive operations go through a `DriveSync` instance:

```go
type DriveSync struct {
    service   *gdrive.Service
    identity  *age.HybridIdentity
    notesDir  string
    configDir string
}
```

It holds the authenticated Google Drive service, the user's encryption identity (for decrypting stored tokens), and the local directory paths.

### OAuth2 Setup Flow

The `Setup` function guides the user through a one-time Google sign-in:

```go
func Setup(configDir string, identity *age.HybridIdentity) error {
    fmt.Println("Google Drive Setup")
    fmt.Println("==================")
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

    // Encrypt and store both the token and the client credentials
    tokenData, _ := json.Marshal(token)
    encPath := filepath.Join(configDir, "google-credentials.json.age")
    crypto.EncryptToFile(encPath, tokenData, identity.Recipient())

    credEncPath := filepath.Join(configDir, "google-client.json.age")
    crypto.EncryptToFile(credEncPath, credData, identity.Recipient())

    fmt.Println("\nDrive setup complete! Credentials stored encrypted.")
    return nil
}
```

Both the OAuth token and the client credentials are encrypted with the user's age identity before being saved. This means even if someone reads the config directory, they can't steal the Google tokens.

### Local HTTP Callback Server

The OAuth flow uses a local HTTP server to catch Google's callback:

```go
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
            return
        }
        code := r.URL.Query().Get("code")
        codeCh <- code
        fmt.Fprint(w, "<html><body><h2>Authentication successful!</h2>...</body></html>")
    })

    srv := &http.Server{Handler: mux}
    go srv.Serve(listener)
    defer srv.Close()

    authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
    openBrowser(authURL)

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
```

Security details:
- **Random port** (`127.0.0.1:0`) -- the OS assigns an available port, avoiding conflicts.
- **CSRF state parameter** -- a 16-byte random hex string is passed to Google and verified on callback. If the state doesn't match, the flow is rejected.
- **5-minute timeout** -- if the user doesn't complete sign-in within 5 minutes, the flow aborts instead of hanging forever.

The `randomState` function generates the CSRF token:

```go
func randomState() (string, error) {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
```

### Persisting Token Source

OAuth tokens expire. The `persistingTokenSource` wraps the standard token source and saves refreshed tokens automatically:

```go
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
            crypto.EncryptToFile(encPath, data, p.identity.Recipient())
        }
    }

    return tok, nil
}
```

This implements the `oauth2.TokenSource` interface. When the base source returns a new access token (meaning a refresh happened), the new token is encrypted and saved. The mutex protects against concurrent refreshes.

### Initializing a Sync Session

The `newDriveSync` function decrypts stored credentials and builds an authenticated client:

```go
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
    json.Unmarshal(tokenData, &token)

    ctx := context.Background()
    pts := &persistingTokenSource{
        base:      oauthConfig.TokenSource(ctx, &token),
        configDir: configDir,
        identity:  identity,
    }
    client := oauth2.NewClient(ctx, pts)

    svc, err := gdrive.NewService(ctx, option.WithHTTPClient(client))
    return &DriveSync{service: svc, identity: identity, notesDir: notesDir, configDir: configDir}, err
}
```

### syncAll -- Full Push

The `syncAll` method walks the local notes directory and uploads every `.age` file:

```go
func (ds *DriveSync) syncAll() error {
    rootID, err := ds.findOrCreateFolder(rootFolderName, "root")
    if err != nil {
        return fmt.Errorf("create root folder: %w", err)
    }

    folderCache := map[string]string{"": rootID}
    var count int

    err = filepath.Walk(ds.notesDir, func(path string, info os.FileInfo, err error) error {
        if info.IsDir() {
            if info.Name() == ".pq-notes" {
                return filepath.SkipDir
            }
            return nil
        }
        if filepath.Ext(path) != ".age" {
            return nil
        }

        relPath, _ := filepath.Rel(ds.notesDir, path)
        parentDir := filepath.Dir(relPath)

        parentID, err := ds.ensureFolderPath(parentDir, rootID, folderCache)
        if err != nil {
            return fmt.Errorf("create folder %s: %w", parentDir, err)
        }

        ds.uploadOrUpdateFile(path, filepath.Base(relPath), parentID)
        count++
        return nil
    })

    fmt.Printf("Sync complete! %d files synced.\n", count)
    return err
}
```

Key behaviors:
- The `.pq-notes` config directory is skipped (it contains keys, not notes).
- Only `.age` files are synced (everything else is ignored).
- The `folderCache` maps relative directory paths to Drive folder IDs, avoiding redundant API calls.

### ensureFolderPath with Cache

The `ensureFolderPath` function creates the entire folder hierarchy on Drive, one level at a time:

```go
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
```

For a path like `work/projects/acme`, this creates three folders (or finds existing ones), caching each ID so that subsequent files in the same directory avoid API calls entirely.

### Pull -- Downloading from Drive

`Pull` is the inverse of Sync. It finds the `pq-notes` folder on Drive (but does not create it -- if it doesn't exist, there's nothing to pull) and downloads missing files:

```go
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
    // ...

    for _, entry := range entries {
        localPath := filepath.Join(notesDir, filepath.FromSlash(entry.RelPath))

        if _, err := os.Stat(localPath); err == nil {
            continue // skip existing files
        }

        os.MkdirAll(filepath.Dir(localPath), 0700)
        resp, _ := ds.service.Files.Get(entry.ID).Download()
        data, _ := io.ReadAll(resp.Body)
        os.WriteFile(localPath, data, 0600)
        restored++
    }
    return nil
}
```

Notice the use of `findFolder` (not `findOrCreateFolder`) -- Pull is read-only on the Drive side.

### Clean -- Removing Orphans

`Clean` builds a set of local file paths, walks the Drive tree, and deletes any Drive file that isn't in the local set:

```go
func Clean(notesDir, configDir string, identity *age.HybridIdentity) error {
    // ... authenticate, find root folder ...

    localFiles := make(map[string]bool)
    filepath.Walk(notesDir, func(path string, info os.FileInfo, err error) error {
        if filepath.Ext(path) == ".age" {
            relPath, _ := filepath.Rel(notesDir, path)
            localFiles[filepath.ToSlash(relPath)] = true
        }
        return nil
    })

    entries, _ := ds.walkDriveTree(rootID, "")

    for _, entry := range entries {
        if localFiles[entry.RelPath] {
            continue
        }
        ds.service.Files.Delete(entry.ID).Do()
        deleted++
    }

    fmt.Printf("Clean complete! %d files removed from Drive.\n", deleted)
    return nil
}
```

### SyncFile and DeleteFile -- TUI Auto-Sync

When Drive auto-sync is enabled in the config, the TUI calls these fire-and-forget functions:

```go
func SyncFile(filePath, notesDir, configDir string, identity *age.HybridIdentity) {
    ds, err := newDriveSync(notesDir, configDir, identity)
    if err != nil {
        log.Printf("drive: sync file: %v", err)
        return
    }
    // ... compute relative path, ensure folders, upload ...
    log.Printf("drive: synced %s", relPath)
}

func DeleteFile(relPath, notesDir, configDir string, identity *age.HybridIdentity) {
    ds, err := newDriveSync(notesDir, configDir, identity)
    if err != nil {
        log.Printf("drive: delete file: %v", err)
        return
    }
    // ... walk Drive tree, find matching file, delete ...
    log.Printf("drive: deleted %s from Drive", relPath)
}
```

Both functions log errors instead of returning them. This is intentional -- they run in the background while the user continues working in the TUI. A sync failure shouldn't crash the interface or block the user.

## Relationships

- **Config** provides the `DriveAutoSync` flag that the TUI checks before calling `SyncFile`/`DeleteFile`.
- **Crypto** encrypts and decrypts the stored OAuth credentials and client config.
- **App (TUI)** calls `SyncFile` after note creation/edit and `DeleteFile` after note deletion, all in goroutines.
- **CLI** exposes `drive setup`, `drive sync`, `drive pull`, and `drive clean` as subcommands.

## Key Takeaways

- **Encrypt credentials at rest** -- OAuth tokens are stored as `.age` files, protected by the same encryption that guards your notes.
- **Use a local callback server for OAuth** -- avoids manual copy-paste of auth codes and provides a smooth browser-based flow.
- **CSRF state validation** prevents redirect attacks during the OAuth flow.
- **Cache folder IDs** to minimize API calls -- the `folderCache` map turns O(n) lookups into O(1) after the first access.
- **Fire-and-forget for auto-sync** -- background sync logs errors instead of returning them, keeping the TUI responsive.

## Next Steps

Your notes are encrypted, organized, searchable, and backed up to the cloud. In the final chapter, we'll add two more capabilities: a **notification daemon** that alerts you when notes are due, and a **sharing system** that lets you securely share notes with contacts using their public keys.
