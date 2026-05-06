package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
	"github.com/kborup-redhat/pq-notes/internal/drive"
)

var driveCmd = &cobra.Command{
	Use:   "drive",
	Short: "Google Drive sync",
}

var driveSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up Google Drive OAuth2 authentication",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return err
		}
		return drive.Setup(configDir, identity)
	},
}

var driveSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manually sync notes to Google Drive",
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir()
		configDir := config.ConfigDirIn(notesDir)
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return err
		}
		return drive.Sync(notesDir, configDir, identity)
	},
}

var driveAutoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Toggle automatic sync after edits",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		cfg.DriveAutoSync = !cfg.DriveAutoSync
		if err := config.Save(cfg, configDir); err != nil {
			return err
		}
		if cfg.DriveAutoSync {
			fmt.Println("Auto-sync enabled")
		} else {
			fmt.Println("Auto-sync disabled")
		}
		return nil
	},
}

func init() {
	driveCmd.AddCommand(driveSetupCmd, driveSyncCmd, driveAutoCmd)
	rootCmd.AddCommand(driveCmd)
}
