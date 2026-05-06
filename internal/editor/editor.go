package editor

import (
	"os"
	"os/exec"
	"strings"

	"filippo.io/age"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

// BuildCommand returns the command name and arguments for opening a file in the specified editor.
// For VS Code ("code"), it adds the --wait flag to block until the editor closes.
// For all other editors, it simply passes the file path.
func BuildCommand(editor, filePath string) (string, []string) {
	if editor == "code" {
		return "code", []string{"--wait", filePath}
	}
	return editor, []string{filePath}
}

// Open opens the specified file in the given editor.
// It connects stdin, stdout, and stderr to the terminal so the user can interact with the editor.
func Open(editor, filePath string) error {
	cmd, args := BuildCommand(editor, filePath)

	command := exec.Command(cmd, args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command.Run()
}

// OpenEncrypted decrypts an .md.age file to a temp file, opens the temp file
// in the editor, re-encrypts the edited content back to the original path,
// and securely deletes the temp file.
func OpenEncrypted(editorName, encryptedPath string, identity *age.HybridIdentity) error {
	plaintext, err := crypto.DecryptFile(encryptedPath, identity)
	if err != nil {
		return err
	}

	base := strings.TrimSuffix(strings.ReplaceAll(encryptedPath, string(os.PathSeparator), "_"), ".age")
	tmpFile, err := os.CreateTemp("", base+".tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer secureDelete(tmpPath)

	if _, err := tmpFile.Write(plaintext); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	if err := Open(editorName, tmpPath); err != nil {
		return err
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return err
	}

	return crypto.EncryptToFile(encryptedPath, edited, identity.Recipient())
}

func secureDelete(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		os.Remove(path)
		return
	}
	zeros := make([]byte, info.Size())
	f.Write(zeros)
	f.Sync()
	f.Close()
	os.Remove(path)
}
