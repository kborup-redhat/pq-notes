package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/share"
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage contacts for note sharing",
}

var contactsAddCmd = &cobra.Command{
	Use:   "add [name] [public-key]",
	Short: "Add a contact",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactsFile := filepath.Join(config.ConfigDirIn(config.NotesDir()), "contacts.yaml")
		if err := share.AddContact(contactsFile, args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Added contact %q\n", args[0])
		return nil
	},
}

var contactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		contactsFile := filepath.Join(config.ConfigDirIn(config.NotesDir()), "contacts.yaml")
		contacts, err := share.LoadContacts(contactsFile)
		if err != nil {
			return err
		}
		if len(contacts) == 0 {
			fmt.Println("No contacts saved")
			return nil
		}
		for _, c := range contacts {
			fmt.Printf("  %s: %s\n", c.Name, c.PublicKey)
		}
		return nil
	},
}

var contactsRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactsFile := filepath.Join(config.ConfigDirIn(config.NotesDir()), "contacts.yaml")
		if err := share.RemoveContact(contactsFile, args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed contact %q\n", args[0])
		return nil
	},
}

func init() {
	contactsCmd.AddCommand(contactsAddCmd, contactsListCmd, contactsRemoveCmd)
	rootCmd.AddCommand(contactsCmd)
}
