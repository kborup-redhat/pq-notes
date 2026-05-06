package share

import (
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
	"gopkg.in/yaml.v3"
)

type Contact struct {
	Name      string `yaml:"name"`
	PublicKey string `yaml:"public_key"`
}

func LoadContacts(path string) ([]Contact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var contacts []Contact
	if err := yaml.Unmarshal(data, &contacts); err != nil {
		return nil, err
	}
	return contacts, nil
}

func SaveContacts(path string, contacts []Contact) error {
	data, err := yaml.Marshal(contacts)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func AddContact(path, name, publicKey string) error {
	recipients, err := age.ParseRecipients(strings.NewReader(publicKey))
	if err != nil || len(recipients) == 0 {
		return fmt.Errorf("invalid public key: must be a valid age recipient")
	}

	contacts, err := LoadContacts(path)
	if err != nil {
		return err
	}
	for _, c := range contacts {
		if c.Name == name {
			return fmt.Errorf("contact %q already exists", name)
		}
	}
	contacts = append(contacts, Contact{Name: name, PublicKey: publicKey})
	return SaveContacts(path, contacts)
}

func RemoveContact(path, name string) error {
	contacts, err := LoadContacts(path)
	if err != nil {
		return err
	}
	filtered := make([]Contact, 0, len(contacts))
	for _, c := range contacts {
		if c.Name != name {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == len(contacts) {
		return fmt.Errorf("contact %q not found", name)
	}
	return SaveContacts(path, filtered)
}

func FindContact(path, name string) (*Contact, error) {
	contacts, err := LoadContacts(path)
	if err != nil {
		return nil, err
	}
	for _, c := range contacts {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("contact %q not found", name)
}
