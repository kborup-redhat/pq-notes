# pq-notes — Design Specification

**Date:** 2026-05-06
**Status:** Approved

## Overview

`pq-notes` is a cross-platform terminal-based notes management system built in Go. It provides a split-pane TUI for creating, organizing, and managing notes with post-quantum encryption, Google Drive sync, note sharing, and a background notification daemon.

## Tech Stack

- **Language:** Go
- **TUI:** Bubble Tea (event loop, layout), Glamour (markdown rendering), Lip Gloss (styling/borders)
- **Encryption:** `filippo.io/age` with post-quantum hybrid keys (X25519 + ML-KEM-768) via `-pq` flag
- **Cloud sync:** Google Drive API (direct OAuth2 integration)
- **Notifications:** OS-native (Linux: `notify-send`, Windows: toast, macOS: `osascript`)
- **Daemon:** systemd user service on Linux, scheduled task on Windows, launchd on macOS
- **Holidays:** `github.com/rickar/cal/v2` (holiday definitions by ISO country code, custom work day support)

## CLI Commands

| Command | Purpose |
|---------|---------|
| `pq-notes` | Launch the TUI |
| `pq-notes daemon` | Run notification service |
| `pq-notes daemon install` | Set up auto-start (systemd / scheduled task / launchd) |
| `pq-notes daemon uninstall` | Remove auto-start |
| `pq-notes daemon status` | Check daemon status |
| `pq-notes key show` | Display your public recipient key |
| `pq-notes key export` | Export public key to file |
| `pq-notes key export --private` | Export private identity (with warning) |
| `pq-notes key import <file>` | Import an existing identity |
| `pq-notes contacts add <name> <key>` | Save a contact's public key |
| `pq-notes contacts list` | List saved contacts |
| `pq-notes contacts remove <name>` | Remove a contact |
| `pq-notes share <note> --for <contact>` | Export encrypted note for sharing |
| `pq-notes import <file>` | Import a shared note |
| `pq-notes drive setup` | Google Drive OAuth2 setup |
| `pq-notes drive sync` | Manual sync to Drive |
| `pq-notes drive auto` | Enable auto-sync after edits |
| `pq-notes config` | Reopen the setup wizard to change any setting |
| `pq-notes config holidays add <name> <DD-MM>` | Add a custom recurring holiday |
| `pq-notes config holidays list` | Show configured holidays |
| `pq-notes config holidays remove <name>` | Remove a custom holiday |

## Storage Layout

```
~/notes/                              (Linux/macOS)
%USERPROFILE%\notes\                  (Windows)
├── Acme-Corp/
│   ├── 2026-05-06-kickoff-meeting.md.age
│   └── 2026-05-07-api-review.md.age
├── Red-Hat/
│   └── 2026-05-06-sprint-planning.md.age
├── .pq-notes/
│   ├── config.yaml                   # user preferences
│   ├── key.txt                       # age identity (private key)
│   ├── contacts.yaml                 # saved public keys
│   ├── notified.json                 # notification tracking
│   ├── google-credentials.json.age   # encrypted OAuth creds
│   └── exports/                      # shared note exports
```

- All notes encrypted with `age` post-quantum hybrid keys
- Customer folders auto-created from customer name input
- Customer name sanitized: spaces to hyphens, special characters removed
- Filename format: `YYYY-MM-DD-<slug>.md.age`
- All paths handled via Go `filepath` for cross-platform compatibility

## Date Format

- **Default:** EU format `DD-MM-YYYY` (e.g. `06-05-2026`)
- **Configurable:** `date_format: eu` or `date_format: us` in `config.yaml`
- US format: `MM-DD-YYYY`
- Input parsing accepts both formats; configured default resolves ambiguity
- Flexible natural language input: `tomorrow`, `friday`, `next week`, `none`

## Note Types

Four built-in note types, each with a type-specific template.

### 1. Meeting Notes (`meeting`)

```yaml
---
customer: Acme Corp
type: meeting
created: 06-05-2026 09:30
due:
tags: []
status: open
attendees: []
---
# Meeting Title

## Agenda
-

## Notes
-

## Action Items
- [ ]
```

### 2. Task (`task`)

```yaml
---
customer: Acme Corp
type: task
created: 06-05-2026 09:30
due: 10-05-2026 17:00
tags: []
status: open
priority: normal
---
# Task Title

## Description

## Acceptance Criteria
- [ ]

## Notes
```

Priority values: `low`, `normal`, `high`, `urgent`

### 3. Reminder (`reminder`)

```yaml
---
customer: Acme Corp
type: reminder
created: 06-05-2026 09:30
due: 07-05-2026 09:00
repeat:
tags: []
status: open
---
# Reminder Title
```

Lightweight — title and due date. The daemon uses the title directly as notification text.

### 4. Follow-up (`followup`)

```yaml
---
customer: Acme Corp
type: followup
created: 06-05-2026 09:30
due: 08-05-2026 09:00
tags: []
status: open
related: 2026-05-06-kickoff-meeting
---
# Follow-up: Kickoff Meeting

## What was agreed

## What needs to happen
- [ ]

## Status update
```

The `related` field links to the original note's filename slug.

## TUI Layout

### Main Screen — Split Pane

```
┌─── Notes ──────────────────────┬─── Preview ─────────────────────────┐
│ DAILY TASKS (3 due)            │                                     │
│ ──────────────────────────     │  # Kickoff Meeting                  │
│ [OVERDUE] Acme Corp            │                                     │
│   [meeting] API review         │  **Customer:** Acme Corp            │
│   Due: 04-05-2026              │  **Created:** 06-05-2026 09:30      │
│                                │  **Due:** 10-05-2026                │
│ [TODAY] Red Hat                 │  **Tags:** #meeting #followup       │
│   [task] Sprint retro          │                                     │
│   Due: 06-05-2026              │  ---                                │
│                                │                                     │
│ [UPCOMING] Acme Corp           │  ## Agenda                          │
│   [followup] Contract renewal  │  - Timeline for phase 2             │
│   Due: 10-05-2026              │  - Resource allocation              │
│                                │                                     │
│ ──────────────────────────     │                                     │
│ ALL NOTES                      │                                     │
│  > Acme-Corp/ (4 notes)        │                                     │
│  > Red-Hat/ (7 notes)          │                                     │
│  v Internal/ (2 notes)         │                                     │
│    2026-05-06-standup.md.age   │                                     │
│    2026-05-05-ideas.md.age     │                                     │
└────────────────────────────────┴─────────────────────────────────────┘
 [n]ew  [e]dit  [d]ue  [t]ag filter  [s]earch  [m]ark done  [q]uit
```

### Navigation

| Key | Action |
|-----|--------|
| `Up/Down` | Navigate up/down in note list |
| `Left/Right` | Collapse/expand customer folders |
| `Enter` | Open note in editor |
| `n` | New note |
| `d` | Set/change due date |
| `t` | Filter by tag |
| `y` | Filter by note type |
| `s` | Search notes |
| `m` | Toggle done/open status |
| `S` | Share note |
| `a` | Toggle show all (including done) |
| `Tab` | Switch focus between list and preview |
| `q` | Quit |

Arrow keys are the primary and only navigation method. No vim-style keybindings.

### Dashboard View

- Default view on launch
- Shows full backlog sorted by due date: overdue first, then soonest due, then no due date
- Notes grouped by urgency section: OVERDUE, TODAY, UPCOMING
- Below the backlog: all notes grouped by customer folder (collapsible)
- Note type shown as a label next to each note
- Tasks with `priority: urgent` or `high` are highlighted
- By default shows `status: open` only; press `a` to include done notes (dimmed)

## New Note Flow (TUI-driven)

All metadata is collected via TUI widgets before the editor opens. No manual frontmatter editing required.

1. Press `n`
2. **Type selection** — arrow keys + Enter:
   - Meeting / Task / Reminder / Follow-up
3. **Customer name** — text input with autocomplete from existing folders
4. **Title** — text input
5. **Due date** — text input with flexible parsing (`DD-MM-YYYY`, `tomorrow`, `friday`, `none`)
6. **Repeat** — selection, only shown if due date was set:
   - None / Daily / Weekly / Monthly / Custom...
   - Custom accepts free text: `every 2nd-last workday`, `every 2 weeks friday`
7. **Tags** — text input, comma-separated, with autocomplete from existing tags
8. **Priority** — selection, only shown for Task type:
   - Low / Normal / High / Urgent
9. **Attendees** — text input, comma-separated, only shown for Meeting type
10. **Related note** — selection from existing notes, only shown for Follow-up type
11. **Confirmation** — summary of all fields with Create/Cancel
12. Editor opens with frontmatter and template pre-filled

All metadata fields are also editable from the TUI after creation using their respective keybindings.

## Editor Integration

- On first launch, the app asks: "Default editor: vi or code?"
- Saved to `.pq-notes/config.yaml`
- When opening a note:
  1. Note is decrypted to a secure temp file
  2. TUI suspends
  3. Editor launches (for VS Code: `code --wait <file>`)
  4. On editor close: note re-encrypted, temp file securely deleted
  5. TUI resumes and refreshes
- Editor preference can be changed anytime via config

## Encryption

- **Library:** `filippo.io/age` (native Go, compiles into the binary)
- **Key type:** Post-quantum hybrid (X25519 + ML-KEM-768) via `age-keygen -pq`
- **All notes encrypted by default** — stored as `.md.age` on disk
- **Decrypted in memory** for TUI preview, search, and filtering
- **First launch key setup:**
  1. Check for existing `age` keys
  2. If keys exist: list them, let user pick
  3. If no keys: ask for name and email, generate a post-quantum key pair
  4. Store identity in `.pq-notes/key.txt`

### Key Management Commands

| Command | Purpose |
|---------|---------|
| `pq-notes key show` | Display public recipient key |
| `pq-notes key export` | Export public key to file |
| `pq-notes key export --private` | Export private identity (with warning) |
| `pq-notes key import <file>` | Import an existing identity |

## Alarms & Notification Daemon

### Background Daemon

A separate command built into the same binary: `pq-notes daemon`

- Scans `~/notes/` every 5 minutes for notes with due dates
- Fires OS-native notifications when due:
  - Linux: `notify-send`
  - Windows: toast notification
  - macOS: `osascript`
- Tracks sent notifications in `.pq-notes/notified.json` to avoid repeats
- Sends notification at due time, then once daily for overdue items

### Daemon Management

- `pq-notes daemon install` — creates auto-start service:
  - Linux: systemd user service (`~/.config/systemd/user/pq-notes.service`)
  - Windows: scheduled task
  - macOS: launchd plist
- `pq-notes daemon uninstall` — removes the service
- `pq-notes daemon status` — shows if running

### Repeating Schedules

| Input | Meaning |
|-------|---------|
| `every monday` | Every Monday |
| `every 2nd-last workday` | 2nd to last working day of each month |
| `every last workday` | Last working day of each month |
| `every 1st` | 1st of every month |
| `every 2 weeks friday` | Every other Friday |
| `monthly 15` | 15th of every month |
| `daily` | Every day |
| `weekly` | Every week (same day) |

- Repeating notes show on dashboard when their next occurrence is due
- Marking a repeating note as done auto-resets status to `open` and calculates next due date
- Working day calculation excludes configured weekend days and public holidays for the configured country
- Holiday calendar provided by `rickar/cal` v2 with 25+ country definitions
- Weekend days configurable per user (e.g. Friday/Saturday for Saudi Arabia, Saturday/Sunday for Denmark)

## Search & Filtering

### Fuzzy Search (`s`)

- Inline search bar at top of note list
- Searches across: note titles, customer names, note content, tags
- Results update live as you type
- `Enter` to select, `Esc` to cancel

### Tag Filter (`t`)

- Shows list of all tags across notes
- Select one or more to filter
- Stacks with other active filters
- `Esc` to clear

### Type Filter (`y`)

- Filter by note type: Meeting / Task / Reminder / Follow-up
- Stacks with tag filter

### Status

- Default: shows `status: open` notes only
- Press `a` to toggle showing all notes including `status: done`
- Done notes appear dimmed

### Sorting

- Dashboard: sorted by due date (overdue first, soonest due, no due date last)
- Within customer folders: sorted by creation date (newest first)

## Note Sharing

### File-based Export

- `pq-notes share <note> --for <contact>` — re-encrypts note with recipient's public key
- Output: `.md.age` file in `.pq-notes/exports/`
- No email/SMTP — user sends the file however they prefer (email, Slack, Teams, etc.)
- Recipient imports with `pq-notes import <file>`

### Shared Google Drive Folder

- Re-encrypt note for recipient's key and place in a shared Drive folder
- Both parties need each other's `age` public keys
- Shared notes stay synced — edits by either party sync through Drive
- Conflict handling: if changed both locally and on Drive, keeps both versions and flags conflict in TUI

### In TUI

- Press `S` on a note → share dialog
- Shows contacts list, pick recipients
- Choose method: export file or Drive

## Contact Management

| Command | Purpose |
|---------|---------|
| `pq-notes contacts add <name> <key>` | Save a contact's public key |
| `pq-notes contacts list` | List saved contacts |
| `pq-notes contacts remove <name>` | Remove a contact |

Contacts stored in `.pq-notes/contacts.yaml`. Public keys exchanged out-of-band.

## Google Drive Sync

- `pq-notes drive setup` — OAuth2 authentication flow, credentials stored encrypted in `.pq-notes/google-credentials.json.age`
- `pq-notes drive sync` — manual push/pull to configured Drive folder
- `pq-notes drive auto` — enable automatic sync after every note save/edit
- Drive folder structure mirrors local: `Notes/Acme-Corp/2026-05-06-kickoff-meeting.md.age`
- Files uploaded in encrypted form — Drive never sees plaintext
- Conflict handling: both versions kept, conflict flagged in TUI for resolution

## First Launch Setup Wizard

When `pq-notes` is launched for the first time (no `config.yaml` exists), the TUI walks through
all configuration in a step-by-step wizard:

### Step 1/5 — Encryption Key

- Check for existing `age` keys (`age-keygen` identities on the system)
- If keys found: list them, let user pick which one to use
- If no keys found: ask for name and email, generate a post-quantum hybrid key pair
- Key stored in `~/notes/.pq-notes/key.txt`

### Step 2/5 — Editor

- Selection: `vi` or `code (VS Code)`
- Arrow keys to highlight, Enter to confirm

### Step 3/5 — Date Format

- Selection: `EU (DD-MM-YYYY)` or `US (MM-DD-YYYY)`
- Default: EU

### Step 4/5 — Country

- Text input with autocomplete showing ISO 3166-1 alpha-2 codes and country names
- Used to load public holidays from `rickar/cal` v2 and set default weekend days
- Examples: `DK` — Denmark, `SA` — Saudi Arabia, `US` — United States

### Step 5/5 — Weekend Days

- Checkbox list of all 7 days, pre-selected based on country:
  - Most countries: Saturday, Sunday pre-selected
  - `SA`, `AE`, `BH`, `KW`, `OM`, `QA`: Friday, Saturday pre-selected
  - `IL`: Friday, Saturday pre-selected
- User can override by toggling any day
- Enter to confirm

### Completion

- Config saved to `~/notes/.pq-notes/config.yaml`
- Key saved to `~/notes/.pq-notes/key.txt`
- Directory structure created
- TUI launches with empty dashboard

### Reconfiguration

`pq-notes config` reopens the setup wizard at any time to change any setting.

## Configuration

Stored in `~/notes/.pq-notes/config.yaml`:

```yaml
editor: vi                        # vi | code
date_format: eu                   # eu | us
country: DK                       # ISO 3166-1 alpha-2 code
weekend: [saturday, sunday]       # configurable weekend days
custom_holidays: []               # user-defined recurring holidays
drive_auto_sync: false
```

### Custom Holidays

Users can add company-specific or personal holidays that are not in the country calendar:

- `pq-notes config holidays add "Company Day" 15-06` — adds a recurring custom holiday (DD-MM)
- `pq-notes config holidays list` — shows all configured holidays (country + custom)
- `pq-notes config holidays remove "Company Day"` — removes a custom holiday

Custom holidays are stored in `config.yaml` and are treated identically to country holidays
for working day calculations and repeating schedule resolution.

## Cross-Platform Support

| Concern | Linux | Windows | macOS |
|---------|-------|---------|-------|
| Notes directory | `~/notes/` | `%USERPROFILE%\notes\` | `~/notes/` |
| Notifications | `notify-send` | Toast notifications | `osascript` |
| Daemon auto-start | systemd user service | Scheduled task | launchd plist |
| Editor (vi) | System vi/vim | Git Bash vi or gvim | System vi/vim |
| Editor (code) | `code --wait` | `code --wait` | `code --wait` |
| Paths | Go `filepath` | Go `filepath` | Go `filepath` |

## Excluded (YAGNI)

- No git integration
- No SMTP / built-in email sending
- No custom note templates beyond the four built-in types
- No multi-user real-time collaboration (sharing is async via file export or Drive)
- No public holiday API integration (uses built-in `rickar/cal` definitions + custom holidays)
