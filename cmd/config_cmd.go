package cmd

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/tui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Reopen the setup wizard to change settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir()
		configDir := config.ConfigDirIn(notesDir)
		setup := tui.NewSetupModel(notesDir, configDir)
		p := tea.NewProgram(setup)
		_, err := p.Run()
		return err
	},
}

var holidaysCmd = &cobra.Command{
	Use:   "holidays",
	Short: "Manage custom holidays",
}

var holidaysAddCmd = &cobra.Command{
	Use:   "add [name] [DD-MM]",
	Short: "Add a custom recurring holiday",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		cfg.CustomHolidays = append(cfg.CustomHolidays, config.CustomHoliday{
			Name: args[0],
			Date: args[1],
		})
		if err := config.Save(cfg, configDir); err != nil {
			return err
		}
		fmt.Printf("Added holiday %q on %s\n", args[0], args[1])
		return nil
	},
}

var holidaysListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show configured holidays",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		if len(cfg.CustomHolidays) == 0 {
			fmt.Println("No custom holidays configured")
			return nil
		}
		for _, h := range cfg.CustomHolidays {
			fmt.Printf("  %s: %s\n", h.Name, h.Date)
		}
		return nil
	},
}

var holidaysRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a custom holiday",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		var filtered []config.CustomHoliday
		for _, h := range cfg.CustomHolidays {
			if h.Name != args[0] {
				filtered = append(filtered, h)
			}
		}
		if len(filtered) == len(cfg.CustomHolidays) {
			fmt.Printf("Holiday %q not found\n", args[0])
			return nil
		}
		cfg.CustomHolidays = filtered
		if err := config.Save(cfg, configDir); err != nil {
			return err
		}
		fmt.Printf("Removed holiday %q\n", args[0])
		return nil
	},
}

func init() {
	holidaysCmd.AddCommand(holidaysAddCmd, holidaysListCmd, holidaysRemoveCmd)
	configCmd.AddCommand(holidaysCmd)
	rootCmd.AddCommand(configCmd)
}
