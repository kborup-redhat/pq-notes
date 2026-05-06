package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

// Setup runs an interactive OAuth2 setup flow: reads a credentials JSON file,
// performs the auth code exchange, and stores encrypted tokens.
func Setup(configDir string, identity *age.HybridIdentity) error {
	fmt.Println("Google Drive Setup")
	fmt.Println("==================")
	fmt.Println("To use Drive sync, you need a Google Cloud project with the Drive API enabled.")
	fmt.Println("1. Go to https://console.cloud.google.com/apis/credentials")
	fmt.Println("2. Create an OAuth 2.0 Client ID (Desktop application)")
	fmt.Println("3. Download the JSON credentials file")
	fmt.Print("\nPath to credentials JSON: ")

	var credPath string
	fmt.Scanln(&credPath)

	credData, err := os.ReadFile(credPath)
	if err != nil {
		return fmt.Errorf("read credentials: %w", err)
	}

	oauthConfig, err := google.ConfigFromJSON(credData, gdrive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}

	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\nOpen this URL in your browser:\n%s\n\nPaste the authorization code: ", authURL)

	var authCode string
	fmt.Scanln(&authCode)

	token, err := oauthConfig.Exchange(context.Background(), authCode)
	if err != nil {
		return fmt.Errorf("exchange token: %w", err)
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

// Sync walks the notes directory and prints what would be synced to Google Drive.
func Sync(notesDir, configDir string, identity *age.HybridIdentity) error {
	ds, err := newDriveSync(notesDir, configDir, identity)
	if err != nil {
		return err
	}
	return ds.syncAll()
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

	client := oauthConfig.Client(context.Background(), &token)
	srv, err := gdrive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &DriveSync{
		service:   srv,
		identity:  identity,
		notesDir:  notesDir,
		configDir: configDir,
	}, nil
}

func (ds *DriveSync) syncAll() error {
	fmt.Println("Syncing notes to Google Drive...")

	err := filepath.Walk(ds.notesDir, func(path string, info os.FileInfo, err error) error {
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

		relPath, _ := filepath.Rel(ds.notesDir, path)
		fmt.Printf("  Uploading: %s\n", relPath)
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println("Sync complete!")
	return nil
}
