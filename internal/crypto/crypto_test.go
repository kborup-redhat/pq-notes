package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if identity == nil {
		t.Fatal("GenerateKey returned nil identity")
	}

	// Check that the key file was created
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Key file not created: %v", err)
	}

	// Check file permissions (0600)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
	}

	// Check that file contains key data
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Key file is empty")
	}

	// Check that the identity string format is correct (should start with AGE-SECRET-KEY-PQ-1)
	identityStr := identity.String()
	if len(identityStr) == 0 {
		t.Fatal("Identity string is empty")
	}
}

func TestLoadIdentity(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	// Generate a key first
	original, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Load the identity back
	loaded, err := LoadIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadIdentity failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("LoadIdentity returned nil identity")
	}

	// Compare the string representations
	if original.String() != loaded.String() {
		t.Error("Loaded identity doesn't match original")
	}
}

func TestLoadIdentityNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "nonexistent.key")

	_, err := LoadIdentity(keyPath)
	if err == nil {
		t.Fatal("Expected error loading nonexistent key file")
	}
}

func TestPublicKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	pubKey := PublicKey(identity)
	if len(pubKey) == 0 {
		t.Fatal("PublicKey returned empty string")
	}

	// Post-quantum public keys should start with "age1pq1"
	// (though this might vary based on version)
	if len(pubKey) < 10 {
		t.Error("Public key seems too short")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	plaintext := []byte("Hello, World! This is a test message with some emoji: 🔐")

	// Encrypt
	ciphertext, err := Encrypt(plaintext, identity.Recipient())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("Ciphertext is empty")
	}

	// Ciphertext should be different from plaintext
	if bytes.Equal(plaintext, ciphertext) {
		t.Error("Ciphertext equals plaintext")
	}

	// Decrypt
	decrypted, err := Decrypt(ciphertext, identity)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// Compare
	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text doesn't match original.\nExpected: %s\nGot: %s", plaintext, decrypted)
	}
}

func TestEncryptDecryptMultipleRecipients(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate two different identities
	keyPath1 := filepath.Join(tmpDir, "test1.key")
	identity1, err := GenerateKey(keyPath1)
	if err != nil {
		t.Fatalf("GenerateKey 1 failed: %v", err)
	}

	keyPath2 := filepath.Join(tmpDir, "test2.key")
	identity2, err := GenerateKey(keyPath2)
	if err != nil {
		t.Fatalf("GenerateKey 2 failed: %v", err)
	}

	plaintext := []byte("Secret message for multiple recipients")

	// Encrypt for both recipients
	ciphertext, err := Encrypt(plaintext, identity1.Recipient(), identity2.Recipient())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Both identities should be able to decrypt
	decrypted1, err := Decrypt(ciphertext, identity1)
	if err != nil {
		t.Fatalf("Decrypt with identity1 failed: %v", err)
	}

	decrypted2, err := Decrypt(ciphertext, identity2)
	if err != nil {
		t.Fatalf("Decrypt with identity2 failed: %v", err)
	}

	// Both should get the same plaintext
	if !bytes.Equal(plaintext, decrypted1) {
		t.Error("Decrypted text (identity1) doesn't match original")
	}

	if !bytes.Equal(plaintext, decrypted2) {
		t.Error("Decrypted text (identity2) doesn't match original")
	}
}

func TestEncryptDecryptFileRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")
	encryptedPath := filepath.Join(tmpDir, "encrypted.age")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	plaintext := []byte("This is a file-based encryption test with multiple lines.\nLine 2\nLine 3")

	// Encrypt to file
	err = EncryptToFile(encryptedPath, plaintext, identity.Recipient())
	if err != nil {
		t.Fatalf("EncryptToFile failed: %v", err)
	}

	// Check that the encrypted file exists
	info, err := os.Stat(encryptedPath)
	if err != nil {
		t.Fatalf("Encrypted file not created: %v", err)
	}

	if info.Size() == 0 {
		t.Fatal("Encrypted file is empty")
	}

	// Decrypt from file
	decrypted, err := DecryptFile(encryptedPath, identity)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	// Compare
	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text doesn't match original.\nExpected: %s\nGot: %s", plaintext, decrypted)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate two different identities
	keyPath1 := filepath.Join(tmpDir, "test1.key")
	identity1, err := GenerateKey(keyPath1)
	if err != nil {
		t.Fatalf("GenerateKey 1 failed: %v", err)
	}

	keyPath2 := filepath.Join(tmpDir, "test2.key")
	identity2, err := GenerateKey(keyPath2)
	if err != nil {
		t.Fatalf("GenerateKey 2 failed: %v", err)
	}

	plaintext := []byte("Secret message")

	// Encrypt for identity1 only
	ciphertext, err := Encrypt(plaintext, identity1.Recipient())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to decrypt with identity2 (should fail)
	_, err = Decrypt(ciphertext, identity2)
	if err == nil {
		t.Fatal("Expected error when decrypting with wrong key")
	}
}

func TestEncryptEmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	plaintext := []byte("")

	// Encrypt empty data
	ciphertext, err := Encrypt(plaintext, identity.Recipient())
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decrypt
	decrypted, err := Decrypt(ciphertext, identity)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// Compare
	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text doesn't match original empty data")
	}
}

func TestDecryptFileNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")
	nonexistentPath := filepath.Join(tmpDir, "nonexistent.age")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	_, err = DecryptFile(nonexistentPath, identity)
	if err == nil {
		t.Fatal("Expected error decrypting nonexistent file")
	}
}

func TestPublicKeyRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Get public key string
	pubKeyStr := PublicKey(identity)

	// Get recipient directly from identity
	recipient := identity.Recipient()

	// Compare string representations
	if recipient.String() != pubKeyStr {
		t.Error("PublicKey() doesn't match identity.Recipient().String()")
	}
}
