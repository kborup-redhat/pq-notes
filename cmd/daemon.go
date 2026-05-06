package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
	"github.com/kborup-redhat/pq-notes/internal/daemon"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the notification daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir()
		configDir := config.ConfigDirIn(notesDir)
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return err
		}
		fmt.Println("pq-notes daemon started")
		daemon.Run(cfg, identity, notesDir, configDir)
		return nil
	},
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install daemon as auto-start service",
	RunE: func(cmd *cobra.Command, args []string) error {
		binary, err := os.Executable()
		if err != nil {
			return err
		}
		if err := daemon.Install(binary); err != nil {
			return err
		}
		fmt.Println("Daemon installed and started")
		return nil
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove daemon auto-start service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.Uninstall(); err != nil {
			return err
		}
		fmt.Println("Daemon uninstalled")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	RunE: func(cmd *cobra.Command, args []string) error {
		status, err := daemon.Status()
		if err != nil {
			fmt.Printf("Daemon status: not running (%v)\n", err)
			return nil
		}
		fmt.Printf("Daemon status: %s", status)
		return nil
	},
}

func init() {
	daemonCmd.AddCommand(daemonInstallCmd, daemonUninstallCmd, daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)
}
