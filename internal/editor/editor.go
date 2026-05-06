package editor

import (
	"os"
	"os/exec"
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
