# pq-notes

A terminal-based notes manager with post-quantum encryption, markdown preview, and Google Drive sync.

All notes are encrypted at rest using [age](https://github.com/FiloSottile/age) with hybrid X25519 + ML-KEM-768 (post-quantum) keys. Notes are stored as encrypted markdown files organized in named folders.

## Features

- **TUI dashboard** — Split-pane layout with note list and live markdown preview
- **Post-quantum encryption** — Every note is encrypted with age hybrid keys (X25519 + ML-KEM-768)
- **Note types** — Meeting notes, tasks, reminders, and follow-ups
- **Due dates and repeating schedules** — Business-day-aware scheduling with country-specific holidays
- **Priority levels** — Low, normal, high, and urgent with visual indicators
- **Tags and filtering** — Tag notes and filter by tag or type
- **Full-text search** — Search across all decrypted note content
- **Google Drive sync** — Encrypted backup to Drive with auto-sync on edit
- **Note sharing** — Share encrypted notes with contacts using their public keys
- **Notification daemon** — Desktop notifications for due and overdue notes
- **Cross-platform** — Linux, macOS, and Windows

## Quick start

```bash
# Download the latest release for your platform from:
# https://github.com/kborup-redhat/pq-notes/releases

# Or build from source:
go install github.com/kborup-redhat/pq-notes@latest

# Run — first launch opens the setup wizard
pq-notes
```

The setup wizard will guide you through choosing an editor (vim/nano/VS Code), date format, and country (for holidays and business days).

## TUI keyboard shortcuts

| Key | Action |
|-----|--------|
| `n` | Create new note |
| `e` / `Enter` | Edit selected note |
| `m` | Toggle done/open status |
| `x` | Delete note (with Drive option if auto-sync is on) |
| `t` | Filter by tag |
| `y` | Filter by type |
| `s` | Search notes |
| `a` | Show all notes (including done) |
| `c` | Show closed notes only |
| `Tab` | Switch focus between list and preview |
| `q` / `Esc` | Quit |

## Note format

Notes are markdown files with YAML frontmatter:

```markdown
---
folder: Acme Corp
type: task
created: 2026-05-06
due: 2026-05-10
repeat: weekly
tags: [billing, urgent]
status: open
priority: high
attendees: [alice, bob]
---

# Meeting agenda

- Review Q2 numbers
- Discuss billing integration
```

## Commands

```
pq-notes                    Launch the TUI
pq-notes config             Reopen setup wizard
pq-notes config holidays    Manage custom holidays (add/list/remove)

pq-notes key show           Display your public key
pq-notes key export         Export public key to file
pq-notes key import         Import an existing identity

pq-notes contacts add       Add a contact's public key
pq-notes contacts list      List contacts
pq-notes contacts remove    Remove a contact

pq-notes share <note>       Share a note with a contact
pq-notes import <file>      Import a shared note

pq-notes drive setup        Set up Google Drive OAuth2
pq-notes drive sync         Manually sync all notes to Drive
pq-notes drive pull         Restore notes from Drive to local
pq-notes drive clean        Remove orphaned files from Drive
pq-notes drive auto         Toggle auto-sync on edit

pq-notes daemon             Run notification daemon
pq-notes daemon install     Install as auto-start service
pq-notes daemon status      Check daemon status
pq-notes daemon uninstall   Remove auto-start service
```

## Google Drive sync

Notes are synced to Drive in their encrypted form — Google never sees plaintext.

```bash
# One-time setup (requires Google Cloud OAuth2 credentials)
pq-notes drive setup

# Sync all notes to Drive
pq-notes drive sync

# Restore notes on a new machine
pq-notes drive pull

# Enable auto-sync (syncs after every edit in the TUI)
pq-notes drive auto

# Clean up orphaned files on Drive
pq-notes drive clean
```

When auto-sync is enabled, deleting a note in the TUI will prompt whether to also delete it from Drive.

## Sharing notes

Share encrypted notes with colleagues using age public keys:

```bash
# Add a contact
pq-notes contacts add alice age1...

# Share a note
pq-notes share ~/notes/acme/meeting.md.age --for alice

# Recipient imports the shared note
pq-notes import meeting.md.age.shared
```

## File structure

```
~/notes/
  <folder>/
    <note>.md.age          # Encrypted note files
  .pq-notes/
    config.yaml            # Configuration
    key.txt                # Private identity (age key)
    contacts.json.age      # Encrypted contacts
    google-client.json.age # Encrypted OAuth2 client config
    google-credentials.json.age # Encrypted OAuth2 tokens
```

## Security

- All notes encrypted at rest with age hybrid keys (X25519 + ML-KEM-768)
- Post-quantum resistant — secure against both classical and quantum attacks
- Private key never leaves the local machine
- Drive sync uploads only encrypted files
- OAuth2 tokens stored encrypted alongside notes
- Contacts stored encrypted

## Building from source

Requires Go 1.26+:

```bash
git clone https://github.com/kborup-redhat/pq-notes.git
cd pq-notes
go build -o pq-notes .
```

## License

See [LICENSE](LICENSE) for details.
