---
title: "Chapter 5: Editor"
order: 5
---

# Chapter 5: Editor Integration

A terminal notes app wouldn't be very useful if you had to manually open each file in your editor. The Editor package bridges the gap between pq-notes and whatever text editor the user prefers — whether that's Vim, Nano, VS Code, or anything else.

Think of this package as a "please open this file" button that knows the quirks of different editors.

## How It Works

The package provides two functions:

1. **BuildCommand** — constructs the command name and arguments for a given editor
2. **Open** — executes the command and waits for the editor to close

The key insight is that most editors work the same way (`editor filename`), but VS Code needs a special `--wait` flag to block the calling process until the user closes the file.

## Code Deep Dive

### Building the Command

```go
func BuildCommand(editor, filePath string) (string, []string) {
    if editor == "code" {
        return "code", []string{"--wait", filePath}
    }
    return editor, []string{filePath}
}
```

This function returns the command and arguments separately — matching what Go's `exec.Command` expects. The separation makes it easy to test without actually launching an editor.

**Why does VS Code need `--wait`?** Most terminal editors (Vim, Nano, Emacs) take over the terminal and block until you quit them. VS Code, however, is a GUI application. When you run `code file.txt`, it opens the file in an already-running VS Code window and the `code` command returns immediately. The `--wait` flag tells VS Code to keep the command running until you close that specific file tab, allowing pq-notes to wait for the user to finish editing.

### Opening a File

```go
func Open(editor, filePath string) error {
    cmd, args := BuildCommand(editor, filePath)

    command := exec.Command(cmd, args...)
    command.Stdin = os.Stdin
    command.Stdout = os.Stdout
    command.Stderr = os.Stderr

    return command.Run()
}
```

The three `os.Std*` assignments are crucial:

- **`command.Stdin = os.Stdin`** — lets the editor read keyboard input from the terminal
- **`command.Stdout = os.Stdout`** — lets the editor draw its UI to the terminal
- **`command.Stderr = os.Stderr`** — lets the editor display error messages

Without these, a terminal-based editor like Vim would have no way to display its interface or read keystrokes. The editor process inherits the parent's terminal, effectively "taking over" the screen until the user saves and quits.

`command.Run()` blocks until the editor process exits. This is what we want — the note needs to be fully edited before pq-notes can read back the content, re-encrypt it, and save it.

### How It Fits in the Workflow

The editor is used in a decrypt-edit-encrypt cycle:

1. **NoteStore** decrypts a note to a temporary plaintext file
2. **Editor.Open** launches the user's editor on that temp file
3. The user edits, saves, and closes the editor
4. **NoteStore** reads the modified file, parses it, and re-encrypts it
5. The temp file is deleted

This approach means plaintext only exists on disk briefly — in a temporary file during editing. The rest of the time, notes are encrypted.

## Relationships

- **Config** provides the `Editor` setting (`"code"`, `"vim"`, `"nano"`, etc.) that determines which editor to launch.
- **NoteStore** orchestrates the decrypt-edit-encrypt workflow, calling `editor.Open` as the middle step.

The Editor package has no dependencies on other internal packages — just Go's standard library.

## Key Takeaways

- **`exec.Command`** runs external processes. Separate the command name from its arguments.
- **Connect stdin/stdout/stderr** when launching interactive terminal programs, or they won't be able to interact with the user.
- **`Run()`** blocks until the process exits — the right choice when you need to wait for user action.
- **Editor-specific flags** (like VS Code's `--wait`) are a common pattern when integrating with external tools. A simple `if` handles the edge case cleanly.
- **Separating command construction from execution** (`BuildCommand` vs `Open`) makes the logic testable without side effects.

## Next Steps

We've now covered all the building blocks: configuration, encryption, note modeling, date handling, and editor integration. In the next chapter, we'll see how the NoteStore ties all of these together into a complete encrypted CRUD layer.
