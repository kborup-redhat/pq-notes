package share

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

func generateTestPublicKey(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.txt")
	identity, err := crypto.GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return crypto.PublicKey(identity)
}

func TestContactsCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	contactsFile := filepath.Join(tmpDir, "contacts.yaml")

	aliceKey := generateTestPublicKey(t)
	bobKey := generateTestPublicKey(t)

	if err := AddContact(contactsFile, "Alice", aliceKey); err != nil {
		t.Fatalf("AddContact: %v", err)
	}

	contacts, err := LoadContacts(contactsFile)
	if err != nil {
		t.Fatalf("LoadContacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].Name != "Alice" {
		t.Errorf("expected name Alice, got %s", contacts[0].Name)
	}

	if err := AddContact(contactsFile, "Bob", bobKey); err != nil {
		t.Fatal(err)
	}
	contacts, _ = LoadContacts(contactsFile)
	if len(contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(contacts))
	}

	if err := RemoveContact(contactsFile, "Alice"); err != nil {
		t.Fatal(err)
	}
	contacts, _ = LoadContacts(contactsFile)
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact after removal, got %d", len(contacts))
	}
	if contacts[0].Name != "Bob" {
		t.Errorf("expected Bob remaining, got %s", contacts[0].Name)
	}
}

func TestDuplicateContact(t *testing.T) {
	tmpDir := t.TempDir()
	contactsFile := filepath.Join(tmpDir, "contacts.yaml")

	key1 := generateTestPublicKey(t)
	key2 := generateTestPublicKey(t)

	if err := AddContact(contactsFile, "Alice", key1); err != nil {
		t.Fatalf("first AddContact: %v", err)
	}

	err := AddContact(contactsFile, "Alice", key2)
	if err == nil {
		t.Fatal("expected error when adding duplicate contact, got nil")
	}

	contacts, err := LoadContacts(contactsFile)
	if err != nil {
		t.Fatalf("LoadContacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact after duplicate attempt, got %d", len(contacts))
	}
	if contacts[0].PublicKey != key1 {
		t.Errorf("expected original key, got %s", contacts[0].PublicKey)
	}
}

func TestFindContact(t *testing.T) {
	tmpDir := t.TempDir()
	contactsFile := filepath.Join(tmpDir, "contacts.yaml")

	aliceKey := generateTestPublicKey(t)
	bobKey := generateTestPublicKey(t)

	if err := AddContact(contactsFile, "Alice", aliceKey); err != nil {
		t.Fatalf("AddContact: %v", err)
	}
	if err := AddContact(contactsFile, "Bob", bobKey); err != nil {
		t.Fatalf("AddContact: %v", err)
	}

	contact, err := FindContact(contactsFile, "Bob")
	if err != nil {
		t.Fatalf("FindContact Bob: %v", err)
	}
	if contact.Name != "Bob" {
		t.Errorf("expected name Bob, got %s", contact.Name)
	}
	if contact.PublicKey != bobKey {
		t.Errorf("expected Bob's key, got %s", contact.PublicKey)
	}

	_, err = FindContact(contactsFile, "Charlie")
	if err == nil {
		t.Fatal("expected error when finding non-existent contact, got nil")
	}
}

func TestRemoveContactNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	contactsFile := filepath.Join(tmpDir, "contacts.yaml")

	key := generateTestPublicKey(t)
	if err := AddContact(contactsFile, "Alice", key); err != nil {
		t.Fatalf("AddContact: %v", err)
	}

	err := RemoveContact(contactsFile, "NonExistent")
	if err == nil {
		t.Fatal("expected error when removing non-existent contact, got nil")
	}
}

func TestAddContactInvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	contactsFile := filepath.Join(tmpDir, "contacts.yaml")

	err := AddContact(contactsFile, "Alice", "not-a-valid-key")
	if err == nil {
		t.Fatal("expected error when adding contact with invalid key, got nil")
	}
}

func TestLoadContactsNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	contactsFile := filepath.Join(tmpDir, "does-not-exist.yaml")

	contacts, err := LoadContacts(contactsFile)
	if err != nil {
		t.Fatalf("LoadContacts on missing file should not error: %v", err)
	}
	if contacts != nil {
		t.Fatalf("expected nil contacts for missing file, got %v", contacts)
	}
}

func TestShareAndImport(t *testing.T) {
	tmpDir := t.TempDir()

	senderKeyPath := filepath.Join(tmpDir, "sender-key.txt")
	senderIdentity, err := crypto.GenerateKey(senderKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	recipientKeyPath := filepath.Join(tmpDir, "recipient-key.txt")
	recipientIdentity, err := crypto.GenerateKey(recipientKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("---\ncustomer: Test\ntype: task\n---\n# Shared Note")
	notePath := filepath.Join(tmpDir, "original.md.age")
	if err := crypto.EncryptToFile(notePath, plaintext, senderIdentity.Recipient()); err != nil {
		t.Fatal(err)
	}

	exportDir := filepath.Join(tmpDir, "exports")
	os.MkdirAll(exportDir, 0700)

	exportPath, err := ShareNote(notePath, senderIdentity, recipientIdentity.Recipient(), exportDir)
	if err != nil {
		t.Fatalf("ShareNote: %v", err)
	}

	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("export file not created: %v", err)
	}

	imported, err := crypto.DecryptFile(exportPath, recipientIdentity)
	if err != nil {
		t.Fatalf("recipient cannot decrypt shared note: %v", err)
	}
	if string(imported) != string(plaintext) {
		t.Errorf("imported content mismatch")
	}
}
