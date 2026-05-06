package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name     string
		editor   string
		filePath string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "VS Code editor",
			editor:   "code",
			filePath: "/tmp/test.txt",
			wantCmd:  "code",
			wantArgs: []string{"--wait", "/tmp/test.txt"},
		},
		{
			name:     "vi editor",
			editor:   "vi",
			filePath: "/tmp/test.txt",
			wantCmd:  "vi",
			wantArgs: []string{"/tmp/test.txt"},
		},
		{
			name:     "vim editor",
			editor:   "vim",
			filePath: "/home/user/notes.txt",
			wantCmd:  "vim",
			wantArgs: []string{"/home/user/notes.txt"},
		},
		{
			name:     "nano editor",
			editor:   "nano",
			filePath: "/var/tmp/note.md",
			wantCmd:  "nano",
			wantArgs: []string{"/var/tmp/note.md"},
		},
		{
			name:     "empty editor defaults to code behavior",
			editor:   "",
			filePath: "/tmp/file.txt",
			wantCmd:  "",
			wantArgs: []string{"/tmp/file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotArgs := BuildCommand(tt.editor, tt.filePath)

			if gotCmd != tt.wantCmd {
				t.Errorf("BuildCommand() command = %v, want %v", gotCmd, tt.wantCmd)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("BuildCommand() args length = %d, want %d", len(gotArgs), len(tt.wantArgs))
				return
			}

			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("BuildCommand() args[%d] = %v, want %v", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestOpenEncrypted_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.txt")

	identity, err := crypto.GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	original := []byte("---\ncustomer: acme\n---\n\n# Test Note\n\nHello world\n")
	encPath := filepath.Join(dir, "note.md.age")
	if err := crypto.EncryptToFile(encPath, original, identity.Recipient()); err != nil {
		t.Fatalf("EncryptToFile: %v", err)
	}

	// "true" exits immediately without modifying the file, so content should survive unchanged
	if err := OpenEncrypted("true", encPath, identity); err != nil {
		t.Fatalf("OpenEncrypted: %v", err)
	}

	decrypted, err := crypto.DecryptFile(encPath, identity)
	if err != nil {
		t.Fatalf("DecryptFile after edit: %v", err)
	}

	if string(decrypted) != string(original) {
		t.Errorf("content changed unexpectedly:\ngot:  %q\nwant: %q", decrypted, original)
	}

	// Verify temp file was cleaned up
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tmp.*"))
	if len(matches) > 0 {
		t.Errorf("temp files not cleaned up: %v", matches)
	}

	_ = os.Remove(keyPath)
}
