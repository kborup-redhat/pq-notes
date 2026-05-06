package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
	internalShare "github.com/kborup-redhat/pq-notes/internal/share"
)

var shareCmd = &cobra.Command{
	Use:   "share [note-path]",
	Short: "Export an encrypted note for sharing",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactName, _ := cmd.Flags().GetString("for")
		if contactName == "" {
			return fmt.Errorf("--for flag is required (contact name)")
		}

		configDir := config.ConfigDirIn(config.NotesDir())
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return fmt.Errorf("load identity: %w", err)
		}

		contact, err := internalShare.FindContact(filepath.Join(configDir, "contacts.yaml"), contactName)
		if err != nil {
			return err
		}

		// Parse the recipient public key
		recipients, err := age.ParseRecipients(strings.NewReader(contact.PublicKey))
		if err != nil {
			return fmt.Errorf("parse recipient key: %w", err)
		}
		if len(recipients) == 0 {
			return fmt.Errorf("no valid recipient key found")
		}

		exportDir := filepath.Join(configDir, "exports")
		os.MkdirAll(exportDir, 0700)

		exportPath, err := internalShare.ShareNote(args[0], identity, recipients[0], exportDir)
		if err != nil {
			return err
		}

		fmt.Printf("Note exported to: %s\n", exportPath)
		fmt.Println("Send this file to your contact via email, Slack, etc.")
		return nil
	},
}

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import a shared note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return err
		}

		plaintext, err := crypto.DecryptFile(args[0], identity)
		if err != nil {
			return fmt.Errorf("cannot decrypt (wrong key?): %w", err)
		}

		filename := filepath.Base(args[0])
		destDir := filepath.Join(config.NotesDir(), "Shared")
		os.MkdirAll(destDir, 0700)
		destPath := filepath.Join(destDir, filename)

		if err := crypto.EncryptToFile(destPath, plaintext, identity.Recipient()); err != nil {
			return err
		}

		fmt.Printf("Note imported to: %s\n", destPath)
		return nil
	},
}

func init() {
	shareCmd.Flags().String("for", "", "Contact name to share with")
	rootCmd.AddCommand(shareCmd, importCmd)
}
