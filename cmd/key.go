package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage encryption keys",
}

var keyShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display your public recipient key",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return fmt.Errorf("load key: %w", err)
		}
		fmt.Println(crypto.PublicKey(identity))
		return nil
	},
}

var keyExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export key to file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		private, _ := cmd.Flags().GetBool("private")

		if private {
			fmt.Fprintln(os.Stderr, "WARNING: This exports your PRIVATE key. Keep it safe!")
			data, err := os.ReadFile(filepath.Join(configDir, "key.txt"))
			if err != nil {
				return err
			}
			outPath := "pq-notes-identity.txt"
			if err := os.WriteFile(outPath, data, 0600); err != nil {
				return err
			}
			fmt.Printf("Private key exported to %s\n", outPath)
		} else {
			identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
			if err != nil {
				return err
			}
			outPath := "pq-notes-public-key.txt"
			if err := os.WriteFile(outPath, []byte(crypto.PublicKey(identity)+"\n"), 0644); err != nil {
				return err
			}
			fmt.Printf("Public key exported to %s\n", outPath)
		}
		return nil
	},
}

var keyImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import an existing identity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(configDir, "key.txt"), data, 0600); err != nil {
			return err
		}
		fmt.Println("Key imported successfully")
		return nil
	},
}

func init() {
	keyExportCmd.Flags().Bool("private", false, "Export private identity (use with caution)")
	keyCmd.AddCommand(keyShowCmd, keyExportCmd, keyImportCmd)
	rootCmd.AddCommand(keyCmd)
}
