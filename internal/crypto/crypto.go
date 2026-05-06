package crypto

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"filippo.io/age"
)

// GenerateKey generates a post-quantum hybrid key pair using age.GenerateHybridIdentity()
// and writes the identity string to keyPath with 0600 permissions.
func GenerateKey(keyPath string) (*age.HybridIdentity, error) {
	identity, err := age.GenerateHybridIdentity()
	if err != nil {
		return nil, fmt.Errorf("failed to generate hybrid identity: %w", err)
	}

	// Write the identity to file with 0600 permissions
	identityStr := identity.String()
	err = os.WriteFile(keyPath, []byte(identityStr+"\n"), 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to write key to file: %w", err)
	}

	return identity, nil
}

// LoadIdentity reads a key file, parses identities with age.ParseIdentities(),
// and returns the first *age.HybridIdentity found.
func LoadIdentity(keyPath string) (*age.HybridIdentity, error) {
	file, err := os.Open(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()

	identities, err := age.ParseIdentities(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse identities: %w", err)
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no identities found in key file")
	}

	// Find the first HybridIdentity
	for _, id := range identities {
		if hybrid, ok := id.(*age.HybridIdentity); ok {
			return hybrid, nil
		}
	}

	return nil, fmt.Errorf("no HybridIdentity found in key file")
}

// PublicKey returns the public key string from a HybridIdentity.
func PublicKey(identity *age.HybridIdentity) string {
	return identity.Recipient().String()
}

// Encrypt encrypts plaintext in memory using age.Encrypt.
func Encrypt(plaintext []byte, recipients ...age.Recipient) ([]byte, error) {
	if len(recipients) == 0 {
		return nil, fmt.Errorf("no recipients provided")
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipients...)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	_, err = w.Write(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to write plaintext: %w", err)
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to finalize encryption: %w", err)
	}

	return buf.Bytes(), nil
}

// Decrypt decrypts ciphertext in memory using age.Decrypt.
func Decrypt(ciphertext []byte, identities ...age.Identity) ([]byte, error) {
	if len(identities) == 0 {
		return nil, fmt.Errorf("no identities provided")
	}

	r, err := age.Decrypt(bytes.NewReader(ciphertext), identities...)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted data: %w", err)
	}

	return plaintext, nil
}

// EncryptToFile encrypts plaintext and writes it to a file.
func EncryptToFile(path string, plaintext []byte, recipients ...age.Recipient) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients provided")
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	w, err := age.Encrypt(file, recipients...)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	_, err = w.Write(plaintext)
	if err != nil {
		return fmt.Errorf("failed to write plaintext: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to finalize encryption: %w", err)
	}

	return nil
}

// DecryptFile reads and decrypts a file.
func DecryptFile(path string, identities ...age.Identity) ([]byte, error) {
	if len(identities) == 0 {
		return nil, fmt.Errorf("no identities provided")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	r, err := age.Decrypt(file, identities...)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted data: %w", err)
	}

	return plaintext, nil
}
