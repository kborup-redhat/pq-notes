---
title: "Chapter 14: Notification Daemon & Sharing"
order: 14
---

# Chapter 14: Notification Daemon & Sharing

Two features complete the pq-notes system. The **notification daemon** runs in the background and taps you on the shoulder when a note is due -- like an alarm clock that reads your calendar. The **sharing system** lets you send encrypted notes to other people -- like sealing a letter in an envelope that only the intended recipient can open.

Together, they transform pq-notes from a personal note-taking tool into a proactive assistant that reminds you of deadlines and collaborates securely with others.

## How It Works

### Daemon Overview

The daemon is a long-running background process that:

1. Loads all notes every 5 minutes.
2. Checks each note's due date against the current time.
3. Sends a desktop notification for notes that are due and haven't been notified today.
4. Tracks which notes have been notified (and when) in a JSON file.

It can be installed as a system service (systemd on Linux, launchd on macOS, scheduled task on Windows) so it starts automatically on login.

### Sharing Overview

The sharing system uses age public-key cryptography:

1. You maintain a contacts list (name + public key) stored in YAML.
2. To share a note, pq-notes decrypts it with your private key, then re-encrypts it with the recipient's public key.
3. The re-encrypted file can be sent to the recipient, who decrypts it with their own private key.

## Code Deep Dive: Notification Daemon

### The Tracker

The `Tracker` persists notification state in a JSON file so the daemon doesn't re-notify for the same note multiple times in one day:

```go
type Tracker struct {
    path     string
    Notified map[string]time.Time `json:"notified"`
}

func NewTracker(path string) *Tracker {
    return &Tracker{
        path:     path,
        Notified: make(map[string]time.Time),
    }
}
```

The `Notified` map uses the note's file path as the key and the last notification time as the value.

### Load and Save

The tracker reads from and writes to a simple JSON file:

```go
func (t *Tracker) Load() error {
    data, err := os.ReadFile(t.path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }
    return json.Unmarshal(data, &t.Notified)
}

func (t *Tracker) Save() error {
    data, err := json.MarshalIndent(t.Notified, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(t.path, data, 0600)
}
```

If the file doesn't exist yet (first run), `Load` returns nil -- the daemon starts with an empty notification history. The `0600` permission on save ensures only the file owner can read the notification log.

### Notification Helpers

Two small methods check and record notification state:

```go
func (t *Tracker) WasNotified(noteID string) bool {
    _, ok := t.Notified[noteID]
    return ok
}

func (t *Tracker) MarkNotified(noteID string) {
    t.Notified[noteID] = time.Now()
}
```

### The ShouldNotify Pure Function

The core decision logic is a pure function with no side effects, making it trivial to test:

```go
func ShouldNotify(due, now time.Time) bool {
    return !due.IsZero() && !due.After(now)
}
```

A note should be notified if it has a due date and that due date is not in the future (i.e., it's due now or overdue).

### The Run Loop

The daemon's main loop runs on a 5-minute ticker:

```go
func Run(cfg *config.Config, identity *age.HybridIdentity, notesDir, configDir string) {
    store := notes.NewNoteStore(notesDir, identity, cfg.DateFormat)
    tracker := NewTracker(filepath.Join(configDir, "notified.json"))
    if err := tracker.Load(); err != nil {
        log.Printf("daemon: load tracker: %v", err)
    }

    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    check(store, tracker)

    for range ticker.C {
        check(store, tracker)
    }
}
```

The function runs `check` immediately on startup (so you get notified right away if something is due), then again every 5 minutes.

### The check Function

Each check cycle lists all notes, filters down to actionable ones, and sends notifications:

```go
func check(store *notes.NoteStore, tracker *Tracker) {
    allNotes, err := store.List()
    if err != nil {
        log.Printf("daemon: list notes: %v", err)
        return
    }

    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

    for _, n := range allNotes {
        if n.Status == notes.StatusDone {
            continue
        }
        if !ShouldNotify(n.Due, now) {
            continue
        }

        noteID := n.FilePath
        lastNotified, ok := tracker.Notified[noteID]
        if ok && lastNotified.After(today) {
            continue
        }

        if err := SendNotification(n.Title, n.Folder, n.Due); err != nil {
            log.Printf("daemon: notify: %v", err)
            continue
        }
        tracker.MarkNotified(noteID)
    }

    if err := tracker.Save(); err != nil {
        log.Printf("daemon: save tracker: %v", err)
    }
}
```

Three filters ensure correct behavior:
1. **Skip done notes** -- completed items don't need reminders.
2. **Skip notes that aren't due yet** -- `ShouldNotify` checks the due date.
3. **Skip already-notified-today** -- compares the last notification time against midnight today. This means you get one notification per note per day, not one every 5 minutes.

### Sending Notifications

The `SendNotification` function formats the message and delegates to a platform-specific function:

```go
func SendNotification(title, folder string, due time.Time) error {
    body := folder + " — Due: " + due.Format("02-01-2006 15:04")
    return sendOSNotification("pq-notes: "+title, body)
}
```

The platform-specific implementations use the native notification system:

**Linux** (via `notify-send`):
```go
//go:build linux

func sendOSNotification(title, body string) error {
    return exec.Command("notify-send", title, body).Run()
}
```

On macOS, the equivalent would use `osascript` to trigger a notification via AppleScript. On Windows, it would use PowerShell's `New-BurntToastNotification` or a similar cmdlet.

### Installing the Daemon

The `Install` function registers the daemon as a system service, with platform detection:

```go
func Install(binaryPath string) error {
    if err := validateBinaryPath(binaryPath); err != nil {
        return err
    }
    switch runtime.GOOS {
    case "linux":
        return installSystemd(binaryPath)
    case "darwin":
        return installLaunchd(binaryPath)
    case "windows":
        return installScheduledTask(binaryPath)
    default:
        return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
    }
}
```

### validateBinaryPath Security

Before embedding the binary path into a service file, the path is validated against injection attacks:

```go
func validateBinaryPath(path string) error {
    for _, ch := range path {
        if ch == '\n' || ch == '\r' {
            return fmt.Errorf("binary path contains invalid characters")
        }
    }
    if strings.ContainsAny(path, "<>&\"'`|;") {
        return fmt.Errorf("binary path contains shell metacharacters")
    }
    return nil
}
```

This prevents an attacker from crafting a path like `/usr/bin/pq-notes; rm -rf /` that could be injected into a systemd unit file or launchd plist. The function rejects newlines, shell metacharacters, and other dangerous characters.

### Systemd Installation (Linux)

The Linux installer creates a systemd user service:

```go
func installSystemd(binaryPath string) error {
    home, _ := os.UserHomeDir()
    dir := filepath.Join(home, ".config", "systemd", "user")
    os.MkdirAll(dir, 0755)

    unit := fmt.Sprintf(
        "[Unit]\nDescription=pq-notes notification daemon\n\n"+
        "[Service]\nExecStart=%s daemon\nRestart=always\n\n"+
        "[Install]\nWantedBy=default.target\n", binaryPath)

    os.WriteFile(filepath.Join(dir, "pq-notes.service"), []byte(unit), 0644)
    exec.Command("systemctl", "--user", "daemon-reload").Run()
    return exec.Command("systemctl", "--user", "enable", "--now", "pq-notes").Run()
}
```

The `--user` flag runs the service in the user's session (not as root). `Restart=always` ensures the daemon restarts if it crashes. `enable --now` both enables on boot and starts immediately.

### Uninstall and Status

```go
func Uninstall() error {
    switch runtime.GOOS {
    case "linux":
        return uninstallSystemd()
    // ... other platforms
    }
}

func Status() (string, error) {
    switch runtime.GOOS {
    case "linux":
        out, err := exec.Command("systemctl", "--user", "is-active", "pq-notes").Output()
        return string(out), err
    default:
        return "unknown", nil
    }
}
```

## Code Deep Dive: Sharing

### The Contact Struct

Contacts are stored as a YAML list with a name and an age public key:

```go
type Contact struct {
    Name      string `yaml:"name"`
    PublicKey string `yaml:"public_key"`
}
```

A contacts file looks like:

```yaml
- name: alice
  public_key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
- name: bob
  public_key: age1...
```

### Loading and Saving Contacts

```go
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
```

Like the notification tracker, a missing file is treated as an empty list rather than an error.

### Adding a Contact with Validation

The `AddContact` function validates the public key before storing it:

```go
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
```

`age.ParseRecipients` is the validation gate. If the key doesn't parse as a valid age recipient, the contact is rejected. This catches typos and malformed keys before they cause errors during actual sharing.

### Removing and Finding Contacts

```go
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
```

`RemoveContact` uses a filter pattern: build a new slice without the target, then check if anything was actually removed by comparing lengths.

### ShareNote -- Decrypt and Re-Encrypt

The `ShareNote` function is the sharing operation itself:

```go
func ShareNote(notePath string, senderIdentity *age.HybridIdentity,
    recipientKey age.Recipient, exportDir string) (string, error) {

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
```

The flow is simple but powerful:
1. **Decrypt** the note using the sender's private key (identity).
2. **Re-encrypt** the plaintext using the recipient's public key.
3. **Write** the re-encrypted file to an export directory.

The original encrypted file is never modified. The exported file has the same filename but is encrypted for a different person. The sender can then transmit this file through any channel (email, messaging, USB drive) -- it's encrypted, so the channel doesn't need to be secure.

## Relationships

- **Daemon** depends on **NoteStore** (to list notes), **Config** (for settings), and **Crypto** (to decrypt stored notes for reading due dates).
- **Sharing** depends on **Crypto** (for `DecryptFile` and `EncryptToFile`) and the **age** library (for recipient key parsing and validation).
- **CLI** exposes daemon commands (`daemon install`, `daemon uninstall`, `daemon status`) and sharing commands (`share add-contact`, `share note`).
- **Config** stores the daemon's notification tracker (`notified.json`) in the config directory alongside other state.

## Key Takeaways

- **Pure functions enable easy testing** -- `ShouldNotify` takes two time values and returns a boolean, with no state or side effects.
- **One notification per day per note** -- the tracker prevents notification fatigue by comparing against midnight today.
- **Validate inputs at the boundary** -- `validateBinaryPath` prevents injection in service files; `age.ParseRecipients` catches invalid keys before they cause downstream errors.
- **Decrypt then re-encrypt for sharing** -- this pattern lets users share encrypted files without ever exposing plaintext to an external system.
- **Platform abstraction via `runtime.GOOS`** -- the same codebase handles systemd, launchd, and Windows scheduled tasks with platform-specific implementations behind a common interface.

## Conclusion

Over these fourteen chapters, we've built a complete understanding of pq-notes -- from the lowest-level cryptographic primitives to the highest-level user experience.

Here's the full architecture at a glance:

| Chapter | Package | What It Does |
|---------|---------|--------------|
| 1 | `config` | YAML settings with OS-aware paths |
| 2 | `crypto` | Post-quantum hybrid encryption (ML-KEM + X25519) |
| 3 | `notes/note` | Note model with frontmatter and Markdown templates |
| 4 | `dateutil` | Flexible date parsing and formatting |
| 5 | `editor` | Terminal editor integration |
| 6 | `notes/store` | Encrypted CRUD orchestration |
| 7 | `calendar` | Business calendar with 40+ countries |
| 8 | `cmd` | Cobra CLI wiring and composition root |
| 9 | `tui` | Bubble Tea application shell and keybindings |
| 10 | `tui/dashboard` | Urgency-grouped dashboard and Glamour preview |
| 11 | `tui/newnote` | Adaptive multi-step note creation wizard |
| 12 | `tui/search+filter` | Full-text search and tag/type filtering |
| 13 | `drive` | Google Drive sync with encrypted OAuth tokens |
| 14 | `daemon` + `share` | Background notifications and public-key sharing |

The design principles that run through every layer:

- **Security by default** -- post-quantum encryption, encrypted credentials, validated inputs at every boundary.
- **Clean separation** -- each package has a single responsibility, dependencies flow downward, and components communicate through well-defined interfaces.
- **Testability** -- pure functions, dependency injection, and sub-models with clear return types make every component independently testable.
- **Platform portability** -- `runtime.GOOS` switches and `filepath.Join` ensure the same codebase runs on Linux, macOS, and Windows.
- **Progressive disclosure** -- from simple note creation to Drive sync to sharing, features build on each other without requiring users to understand everything at once.

Whether you're extending pq-notes with new features or using it as a reference for building your own Go TUI applications, the patterns here -- Bubble Tea sub-models, age encryption, OAuth with local callbacks, systemd integration -- are tools you can apply to any project.
