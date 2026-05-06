---
title: "Chapter 8: Editor Integration"
order: 8
---

# Chapter 8: Editor Integration

A note-taking app is only as good as its editing experience. pq-notes does not try to reinvent text editing -- it launches the user's preferred external editor (VS Code, Vim, Nano, or anything else) and gets out of the way. The challenge is that notes are encrypted on disk, so the editor package must handle a careful decrypt-edit-re-encrypt dance and clean up securely afterward.

Think of the editor integration as a secure courier service. When you want to edit a note, the courier decrypts it into a temporary workspace, lets you work on it, encrypts the result back into the vault, and then shreds the temporary copy so no trace of the plaintext remains.

## How It Works

The editor package has three layers of functionality:

1. **BuildCommand** -- constructs the correct command-line invocation for any editor
2. **Open** -- launches the editor and waits for the user to finish
3. **OpenEncrypted** -- the full secure editing flow: decrypt to temp file, open in editor, re-encrypt, securely delete the temp file

Each layer builds on the one before it, keeping the code simple and testable.

Here is the lifecycle visually:

```
 .md.age file on disk
       |
       v
 [1] Decrypt to memory (plaintext bytes)
       |
       v
 [2] Write to temp file (/tmp/...note.md.tmp.xxxxx)
       |
       v
 [3] Launch editor (vim, code --wait, nano, etc.)
       |   user edits...
       v
 [4] Read edited content from temp file
       |
       v
 [5] Re-encrypt and overwrite original .md.age file
       |
       v
 [6] Secure-delete temp file (zero-wipe + remove)
```

The critical security property: plaintext only exists in two places during editing -- in memory (unavoidable) and in a temporary file (minimized to the editing window). The temp file is securely wiped immediately after use.

Let's look at each layer in detail.

## Code Deep Dive

### BuildCommand -- Editor-Specific Flags

Different editors need different flags. GUI editors like VS Code launch in the background by default, which means the calling process would not know when the user is done editing. The `--wait` flag tells VS Code to block until the editor window is closed:

```go
func BuildCommand(editor, filePath string) (string, []string) {
    if editor == "code" {
        return "code", []string{"--wait", filePath}
    }
    return editor, []string{filePath}
}
```

Terminal editors like Vim and Nano naturally block -- they take over the terminal and only return when the user quits. So the default case simply passes the file path as the only argument.

The function returns the command name and arguments separately, which is exactly what Go's `exec.Command` expects. This separation makes it easy to add support for more editors with special flags in the future (e.g., `emacs` with `--no-splash`, or `subl` with `--wait`).

Why not combine the command and arguments into a single string? Because `exec.Command` in Go does not use a shell to interpret the command. It calls the binary directly with an argument array. Splitting them upfront avoids shell-injection vulnerabilities (a user cannot set their editor to `vim; rm -rf /` and have the second command execute) and keeps the API clean.

### Open -- Launching the Editor

The `Open` function creates a subprocess and connects it to the terminal:

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

The three lines connecting `Stdin`, `Stdout`, and `Stderr` are critical. Without them, the editor subprocess would have no access to the terminal:

- **`Stdin`** -- the editor needs to read keyboard input from the user
- **`Stdout`** -- terminal editors like Vim need to draw their UI to the terminal
- **`Stderr`** -- error messages from the editor should be visible to the user

`command.Run()` blocks until the editor process exits. This is what makes the workflow sequential: create temp file, launch editor, wait, read the result, re-encrypt.

If the editor cannot be found on the system PATH, `exec.Command` returns an error immediately. If the editor crashes, the error is propagated to the caller.

Notice that `Open` does not create any files or handle encryption. It is a general-purpose "launch an editor on a file" function. This makes it reusable -- if pq-notes ever needs to edit an unencrypted file (like the config), it can call `Open` directly without going through the encryption flow.

### OpenEncrypted -- The Full Secure Flow

This is the most important function in the package. It orchestrates the entire decrypt-edit-re-encrypt lifecycle:

```go
func OpenEncrypted(editorName, encryptedPath string, identity *age.HybridIdentity) error {
    plaintext, err := crypto.DecryptFile(encryptedPath, identity)
    if err != nil {
        return err
    }

    base := strings.TrimSuffix(
        strings.ReplaceAll(encryptedPath, string(os.PathSeparator), "_"),
        ".age",
    )
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
```

Let's walk through this step by step:

**Step 1: Decrypt.** The encrypted `.md.age` file is decrypted into a byte slice in memory. At this point, plaintext exists only in RAM.

**Step 2: Create a temp file.** The function creates a temporary file with a name derived from the original path. The path separators are replaced with underscores to create a flat filename (e.g., `notes_acme-corp_2026-05-06-budget.md.tmp.123456`). The `.age` suffix is stripped so the temp file has a `.md` extension, which helps editors apply the correct syntax highlighting. The `*` in the pattern is replaced by `os.CreateTemp` with a random string to avoid collisions.

**Step 3: Defer secure deletion.** The `defer secureDelete(tmpPath)` line ensures the temp file is wiped and removed no matter how the function exits -- whether normally, via an early return on error, or even if a panic occurs. This is the security guarantee.

**Step 4: Write plaintext to the temp file.** The decrypted content is written to the temp file, then the file handle is closed. The editor needs to open the file by path, so we must close our handle first.

**Step 5: Launch the editor.** The user edits the plaintext in their preferred editor. The function blocks until the editor closes.

**Step 6: Read the edited content.** After the editor closes, the (potentially modified) content is read back from the temp file.

**Step 7: Re-encrypt.** The edited content is encrypted using the user's public key and written back to the original `.md.age` path, overwriting the previous encrypted version.

**Step 8: Secure delete (deferred).** The temp file is securely wiped and removed.

### secureDelete -- Wiping Before Removal

A regular `os.Remove` only unlinks the file from the directory. The actual bytes may remain on disk until the operating system reuses those disk sectors. The `secureDelete` function overwrites the file contents with zeros before removing it:

```go
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
```

The function follows a careful sequence:

1. **`os.Stat`** gets the file size. If the file does not exist (already cleaned up), the function returns silently.
2. **`os.OpenFile`** opens the file write-only. If this fails (permissions issue), it falls back to a plain `os.Remove` -- best effort.
3. **`make([]byte, info.Size())`** creates a zero-filled byte slice the same size as the file.
4. **`f.Write(zeros)`** overwrites every byte of the file with zeros.
5. **`f.Sync()`** flushes the write to disk, ensuring the zeros are actually written and not just sitting in an OS buffer.
6. **`f.Close()`** releases the file handle.
7. **`os.Remove(path)`** finally unlinks the file.

This is a single-pass zero-wipe. On modern SSDs with wear-leveling, a single overwrite may not guarantee that the old data is irrecoverable at the hardware level. However, it does prevent casual recovery through filesystem tools, and it is far better than a plain delete. For higher security requirements, you would use a dedicated secure deletion tool or store temp files on an encrypted tmpfs filesystem.

Note that the function never returns an error -- it is purely best-effort. This is appropriate because it runs in a `defer` statement, where the caller has already completed its main work and cannot meaningfully handle a cleanup failure.

### Why Not Use an In-Memory Editor?

You might wonder why pq-notes writes plaintext to a temp file at all. Could it pipe the content to the editor via stdin and read it back from stdout?

In practice, no. Most editors (including VS Code, Vim, and Nano) expect to work with files. They need random access to the content (scrolling, searching, undo), which pipes cannot provide. Some editors support reading from stdin (`vim -`), but the experience is degraded -- features like syntax highlighting based on file extension do not work, and writing back to the original file requires extra configuration.

The temp file approach is a pragmatic trade-off: it enables the full editor experience while keeping the plaintext exposure window as short as possible. The `defer secureDelete` ensures cleanup happens even if things go wrong, and the zero-wipe provides a reasonable level of protection against casual forensics.

### Error Handling Philosophy

The editor package demonstrates two different error handling strategies:

1. **`OpenEncrypted`** -- propagates every error immediately. If decryption fails, temp file creation fails, the editor fails, or re-encryption fails, the error bubbles up to the caller. This is appropriate because each step depends on the previous one succeeding.

2. **`secureDelete`** -- swallows all errors silently. This is appropriate for cleanup code that runs in a `defer`. The main operation is already complete (or already failed). Failing to clean up a temp file is unfortunate but not catastrophic -- and the function still tries its best (falling back to plain `os.Remove` if it cannot open the file for writing).

This distinction -- strict error handling in the main path, best-effort in cleanup -- is a common pattern in Go systems programming.

### The Complete Package at a Glance

The entire editor package is under 90 lines of code. Here is the import block, which reveals all external dependencies:

```go
package editor

import (
    "os"
    "os/exec"
    "strings"

    "filippo.io/age"
    "github.com/kborup-redhat/pq-notes/internal/crypto"
)
```

Only five imports -- three from the standard library, one for the age identity type, and one for the internal crypto package. The editor package does not import `notes`, `config`, `tui`, or any other application package. It knows how to open editors and handle encryption, nothing more. This minimal surface area makes it easy to understand, test, and maintain.

## Relationships

- **Crypto** (`internal/crypto`) provides `DecryptFile` and `EncryptToFile`. The editor package does not know anything about the encryption algorithm -- it just calls these two functions.
- **Config** determines which editor to use. The `Editor` field from the user's config (e.g., `"code"`, `"vim"`, `"nano"`) is passed to `BuildCommand` and `Open`.
- **NoteStore** calls `OpenEncrypted` indirectly through the TUI. After `NoteStore.Create` returns a file path, the TUI passes that path to `OpenEncrypted` so the user can fill in the note's content.
- **TUI** dispatches editor commands as Bubble Tea `Cmd` functions, which run in a separate goroutine. When the editor closes, the TUI receives a message and reloads the note list.

## Key Takeaways

- **Connect `Stdin`, `Stdout`, and `Stderr`** when launching interactive subprocesses. Without this, terminal-based editors cannot function.
- **`defer` guarantees cleanup** even when errors occur or the function exits early. Use it for any resource that must be released.
- **Secure deletion** (overwrite then remove) prevents plaintext from lingering on disk after editing.
- **`f.Sync()`** forces the OS to flush writes to disk. Without it, the zero-overwrite might only exist in memory buffers when `os.Remove` is called.
- **Separate command building from execution** (`BuildCommand` vs `Open`) for testability and extensibility.
- **Keep packages small and focused** -- the entire editor package is under 90 lines with only 5 imports, making it easy to audit for security.

## Next Steps

We now have all the backend pieces: configuration, encryption, note models, scheduling, storage, and editing. In the next chapter, we will bring it all together with the TUI -- the terminal user interface built on Bubble Tea that gives pq-notes its interactive split-pane dashboard.
