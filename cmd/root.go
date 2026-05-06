package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/kborup-redhat/pq-notes/internal/calendar"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
	"github.com/kborup-redhat/pq-notes/internal/notes"
	"github.com/kborup-redhat/pq-notes/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "pq-notes",
	Short: "Post-quantum encrypted terminal notes manager",
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir()
		configDir := config.ConfigDirIn(notesDir)

		if !config.Exists(configDir) {
			if err := os.MkdirAll(configDir, 0700); err != nil {
				return fmt.Errorf("create config dir: %w", err)
			}
			setup := tui.NewSetupModel(notesDir, configDir)
			p := tea.NewProgram(setup)
			if _, err := p.Run(); err != nil {
				return err
			}
		}

		cfg, err := config.Load(configDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		keyPath := filepath.Join(configDir, "key.txt")
		identity, err := crypto.LoadIdentity(keyPath)
		if err != nil {
			return fmt.Errorf("load key: %w", err)
		}

		store := notes.NewNoteStore(notesDir, identity, cfg.DateFormat)
		cal := calendar.New(cfg)

		return tui.RunApp(cfg, store, cal, identity)
	},
}

func Execute() error {
	return rootCmd.Execute()
}
