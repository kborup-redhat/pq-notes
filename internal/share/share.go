package share

import (
	"path/filepath"

	"filippo.io/age"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

func ShareNote(notePath string, senderIdentity *age.HybridIdentity, recipientKey age.Recipient, exportDir string) (string, error) {
	plaintext, err := crypto.DecryptFile(notePath, senderIdentity)
	if err != nil {
		return "", err
	}
	filename := filepath.Base(notePath)
	exportPath := filepath.Join(exportDir, filename)
	if err := crypto.EncryptToFile(exportPath, plaintext, recipientKey); err != nil {
		return "", err
	}
	return exportPath, nil
}
