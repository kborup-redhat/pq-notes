# pq-notes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `pq-notes`, a cross-platform terminal-based notes management system with post-quantum encryption, TUI, notifications, Google Drive sync, and sharing.

**Architecture:** Single Go binary with a Bubble Tea v2 TUI front-end. Notes are stored as age-encrypted markdown files organized in customer folders under `~/notes/`. A background daemon handles notifications. The app is structured as layered packages: `config` (settings/setup), `crypto` (age encryption), `notes` (note CRUD and storage), `calendar` (holidays/workdays), `schedule` (repeating date parsing), `tui` (Bubble Tea UI), `daemon` (notifications), `drive` (Google Drive sync), `share` (contact management and export), and `cmd` (CLI entry point using cobra).

**Tech Stack:** Go 1.24+, charm.land/bubbletea/v2, charm.land/glamour/v2, charm.land/lipgloss/v2, filippo.io/age v1.3+, github.com/rickar/cal/v2, github.com/spf13/cobra, google.golang.org/api (Drive v3)

---

## File Structure

```
terminal-notes/
├── main.go                          # entry point, delegates to cmd
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go                      # cobra root command (launches TUI or setup wizard)
│   ├── daemon.go                    # daemon, daemon install/uninstall/status subcommands
│   ├── key.go                       # key show/export/import subcommands
│   ├── contacts.go                  # contacts add/list/remove subcommands
│   ├── share.go                     # share and import subcommands
│   ├── drive.go                     # drive setup/sync/auto subcommands
│   └── config_cmd.go               # config and config holidays subcommands
├── internal/
│   ├── config/
│   │   ├── config.go               # Config struct, Load, Save, NotesDir, ConfigDir
│   │   └── config_test.go
│   ├── crypto/
│   │   ├── crypto.go               # GenerateKey, LoadIdentity, Encrypt, Decrypt, ExportPublicKey
│   │   └── crypto_test.go
│   ├── notes/
│   │   ├── note.go                 # Note struct, frontmatter parsing, template generation
│   │   ├── store.go                # NoteStore: Create, List, Read, Update, Delete, Search
│   │   └── store_test.go
│   ├── calendar/
│   │   ├── calendar.go             # BusinessCal: wraps rickar/cal, configures weekend/holidays
│   │   └── calendar_test.go
│   ├── schedule/
│   │   ├── schedule.go             # ParseRepeat, NextOccurrence for repeating schedules
│   │   └── schedule_test.go
│   ├── dateutil/
│   │   ├── dateutil.go             # ParseDate (EU/US/natural language), FormatDate
│   │   └── dateutil_test.go
│   ├── editor/
│   │   ├── editor.go               # OpenInEditor: suspends TUI, launches vi/code
│   │   └── editor_test.go
│   ├── daemon/
│   │   ├── daemon.go               # Run loop: scan notes, check due, fire notifications
│   │   ├── notify.go               # OS-native notification dispatch (linux/windows/mac)
│   │   ├── notify_linux.go         # notify-send implementation
│   │   ├── notify_windows.go       # toast implementation
│   │   ├── notify_darwin.go        # osascript implementation
│   │   ├── install.go              # systemd/scheduled-task/launchd service management
│   │   └── daemon_test.go
│   ├── drive/
│   │   ├── drive.go                # OAuth2 setup, sync, auto-sync, conflict detection
│   │   └── drive_test.go
│   ├── share/
│   │   ├── contacts.go             # Contact struct, LoadContacts, SaveContacts
│   │   ├── share.go                # ShareNote (re-encrypt for recipient), ImportNote
│   │   └── share_test.go
│   └── tui/
│       ├── app.go                  # Root model: manages views, global key handling
│       ├── setup.go                # First-launch setup wizard (5-step flow)
│       ├── dashboard.go            # Dashboard view: backlog sorted by due date
│       ├── notelist.go             # Note list component with customer folders
│       ├── preview.go              # Markdown preview pane using Glamour
│       ├── newnote.go              # New note wizard (type, customer, title, due, etc.)
│       ├── search.go               # Fuzzy search overlay
│       ├── filter.go               # Tag and type filter overlays
│       ├── dateinput.go            # Date input widget with flexible parsing
│       ├── sharedialog.go          # Share dialog (pick contacts, method)
│       └── styles.go               # Lip Gloss style definitions
```

---

### Task 1: Project Scaffolding & Go Module

**Files:**
- Create: `main.go`
- Create: `go.mod`
- Create: `cmd/root.go`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go mod init github.com/kborup-redhat/pq-notes
```
Expected: `go.mod` created with module path

- [ ] **Step 2: Create main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/kborup-redhat/pq-notes/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Create cmd/root.go with cobra root command**

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pq-notes",
	Short: "Post-quantum encrypted terminal notes manager",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Will launch TUI or setup wizard
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
```

- [ ] **Step 4: Add cobra dependency and verify build**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go get github.com/spf13/cobra@latest
go build ./...
```
Expected: builds successfully, binary created

- [ ] **Step 5: Run the binary to verify it starts**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go run . --help
```
Expected: shows help text with "Post-quantum encrypted terminal notes manager"

- [ ] **Step 6: Commit**

```bash
git add main.go go.mod go.sum cmd/root.go
git commit -m "feat: scaffold pq-notes project with cobra CLI"
```

---

### Task 2: Configuration Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for config load/save**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Editor:     "vi",
		DateFormat: "eu",
		Country:    "DK",
		Weekend:    []string{"saturday", "sunday"},
	}

	configDir := filepath.Join(tmpDir, ".pq-notes")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := Save(cfg, configDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(configDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Editor != "vi" {
		t.Errorf("expected editor vi, got %s", loaded.Editor)
	}
	if loaded.Country != "DK" {
		t.Errorf("expected country DK, got %s", loaded.Country)
	}
	if len(loaded.Weekend) != 2 || loaded.Weekend[0] != "saturday" {
		t.Errorf("unexpected weekend: %v", loaded.Weekend)
	}
}

func TestNotesDir(t *testing.T) {
	dir := NotesDir()
	if dir == "" {
		t.Error("NotesDir returned empty string")
	}
}

func TestConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	dir := ConfigDirIn(tmpDir)
	expected := filepath.Join(tmpDir, ".pq-notes")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestDefaultWeekendForCountry(t *testing.T) {
	tests := []struct {
		country  string
		expected []string
	}{
		{"DK", []string{"saturday", "sunday"}},
		{"SA", []string{"friday", "saturday"}},
		{"IL", []string{"friday", "saturday"}},
		{"US", []string{"saturday", "sunday"}},
	}
	for _, tt := range tests {
		got := DefaultWeekend(tt.country)
		if len(got) != len(tt.expected) {
			t.Errorf("country %s: expected %v, got %v", tt.country, tt.expected, got)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("country %s: expected %v, got %v", tt.country, tt.expected, got)
				break
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/config/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write config.go implementation**

```go
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type CustomHoliday struct {
	Name string `yaml:"name"`
	Date string `yaml:"date"` // DD-MM format
}

type Config struct {
	Editor         string          `yaml:"editor"`
	DateFormat     string          `yaml:"date_format"`
	Country        string          `yaml:"country"`
	Weekend        []string        `yaml:"weekend"`
	CustomHolidays []CustomHoliday `yaml:"custom_holidays"`
	DriveAutoSync  bool            `yaml:"drive_auto_sync"`
}

func NotesDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("USERPROFILE"), "notes")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "notes")
}

func ConfigDirIn(notesDir string) string {
	return filepath.Join(notesDir, ".pq-notes")
}

func Exists(configDir string) bool {
	_, err := os.Stat(filepath.Join(configDir, "config.yaml"))
	return err == nil
}

func Save(cfg *Config, configDir string) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0600)
}

func Load(configDir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

var fridaySaturdayCountries = map[string]bool{
	"SA": true, "AE": true, "BH": true, "KW": true, "OM": true, "QA": true, "IL": true,
}

func DefaultWeekend(country string) []string {
	if fridaySaturdayCountries[strings.ToUpper(country)] {
		return []string{"friday", "saturday"}
	}
	return []string{"saturday", "sunday"}
}
```

- [ ] **Step 4: Add yaml dependency and run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go get gopkg.in/yaml.v3@latest
go test ./internal/config/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add config package with load/save and country weekend defaults"
```

---

### Task 3: Date Utilities Package

**Files:**
- Create: `internal/dateutil/dateutil.go`
- Create: `internal/dateutil/dateutil_test.go`

- [ ] **Step 1: Write failing tests for date parsing and formatting**

```go
package dateutil

import (
	"testing"
	"time"
)

func TestFormatDate(t *testing.T) {
	dt := time.Date(2026, 5, 6, 9, 30, 0, 0, time.Local)

	eu := FormatDate(dt, "eu")
	if eu != "06-05-2026 09:30" {
		t.Errorf("EU format: expected 06-05-2026 09:30, got %s", eu)
	}

	us := FormatDate(dt, "us")
	if us != "05-06-2026 09:30" {
		t.Errorf("US format: expected 05-06-2026 09:30, got %s", us)
	}
}

func TestFormatDateOnly(t *testing.T) {
	dt := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	eu := FormatDateOnly(dt, "eu")
	if eu != "06-05-2026" {
		t.Errorf("expected 06-05-2026, got %s", eu)
	}
}

func TestParseDateEU(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.Local)
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"06-05-2026", time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)},
		{"06-05-2026 17:00", time.Date(2026, 5, 6, 17, 0, 0, 0, time.Local)},
		{"15-12-2026", time.Date(2026, 12, 15, 0, 0, 0, 0, time.Local)},
	}
	for _, tt := range tests {
		got, err := ParseDate(tt.input, "eu", now)
		if err != nil {
			t.Errorf("ParseDate(%q): %v", tt.input, err)
			continue
		}
		if !got.Equal(tt.expected) {
			t.Errorf("ParseDate(%q): expected %v, got %v", tt.input, tt.expected, got)
		}
	}
}

func TestParseDateUS(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.Local)
	got, err := ParseDate("05-06-2026", "us", now)
	if err != nil {
		t.Fatalf("ParseDate: %v", err)
	}
	expected := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestParseDateNaturalLanguage(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.Local) // Wednesday
	tests := []struct {
		input string
		check func(time.Time) bool
		desc  string
	}{
		{"tomorrow", func(t time.Time) bool { return t.Day() == 7 && t.Month() == 5 }, "should be May 7"},
		{"friday", func(t time.Time) bool { return t.Weekday() == time.Friday }, "should be Friday"},
		{"none", func(t time.Time) bool { return t.IsZero() }, "should be zero"},
	}
	for _, tt := range tests {
		got, err := ParseDate(tt.input, "eu", now)
		if err != nil {
			t.Errorf("ParseDate(%q): %v", tt.input, err)
			continue
		}
		if !tt.check(got) {
			t.Errorf("ParseDate(%q): %s, got %v", tt.input, tt.desc, got)
		}
	}
}

func TestParseDateNone(t *testing.T) {
	now := time.Now()
	got, err := ParseDate("none", "eu", now)
	if err != nil {
		t.Fatalf("ParseDate(none): %v", err)
	}
	if !got.IsZero() {
		t.Errorf("expected zero time for 'none', got %v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/dateutil/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write dateutil.go implementation**

```go
package dateutil

import (
	"fmt"
	"strings"
	"time"
)

func FormatDate(t time.Time, format string) string {
	if t.IsZero() {
		return ""
	}
	if t.Hour() == 0 && t.Minute() == 0 {
		return FormatDateOnly(t, format)
	}
	switch format {
	case "us":
		return t.Format("01-02-2006 15:04")
	default:
		return t.Format("02-01-2006 15:04")
	}
}

func FormatDateOnly(t time.Time, format string) string {
	if t.IsZero() {
		return ""
	}
	switch format {
	case "us":
		return t.Format("01-02-2006")
	default:
		return t.Format("02-01-2006")
	}
}

func ParseDate(input, format string, now time.Time) (time.Time, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" || input == "none" {
		return time.Time{}, nil
	}

	switch input {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	case "tomorrow":
		tmrw := now.AddDate(0, 0, 1)
		return time.Date(tmrw.Year(), tmrw.Month(), tmrw.Day(), 0, 0, 0, 0, now.Location()), nil
	case "next week":
		nw := now.AddDate(0, 0, 7)
		return time.Date(nw.Year(), nw.Month(), nw.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	weekdays := map[string]time.Weekday{
		"monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
		"thursday": time.Thursday, "friday": time.Friday,
		"saturday": time.Saturday, "sunday": time.Sunday,
	}
	if wd, ok := weekdays[input]; ok {
		return nextWeekday(now, wd), nil
	}

	// Try date+time formats
	var layouts []string
	if format == "us" {
		layouts = []string{"01-02-2006 15:04", "01-02-2006"}
	} else {
		layouts = []string{"02-01-2006 15:04", "02-01-2006"}
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, input, now.Location()); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date: %q", input)
}

func nextWeekday(from time.Time, target time.Weekday) time.Time {
	daysAhead := int(target) - int(from.Weekday())
	if daysAhead <= 0 {
		daysAhead += 7
	}
	next := from.AddDate(0, 0, daysAhead)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, from.Location())
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/dateutil/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/dateutil/
git commit -m "feat: add dateutil package with EU/US date parsing and natural language"
```

---

### Task 4: Encryption Package

**Files:**
- Create: `internal/crypto/crypto.go`
- Create: `internal/crypto/crypto_test.go`

- [ ] **Step 1: Write failing tests for key generation, encrypt, decrypt**

```go
package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateAndSaveKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.txt")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	if identity == nil {
		t.Fatal("identity is nil")
	}

	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("key file not created: %v", err)
	}
}

func TestLoadIdentity(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.txt")

	orig, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	loaded, err := LoadIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}

	if orig.Recipient().String() != loaded.Recipient().String() {
		t.Error("loaded identity has different recipient")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.txt")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	plaintext := []byte("# Meeting Notes\n\nThis is a test note.")

	encrypted, err := Encrypt(plaintext, identity.Recipient())
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if bytes.Equal(encrypted, plaintext) {
		t.Error("encrypted data equals plaintext")
	}

	decrypted, err := Decrypt(encrypted, identity)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted data does not match: got %q", decrypted)
	}
}

func TestEncryptDecryptFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.txt")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	plaintext := []byte("---\ncustomer: Acme Corp\n---\n# Test Note")
	notePath := filepath.Join(tmpDir, "test.md.age")

	if err := EncryptToFile(notePath, plaintext, identity.Recipient()); err != nil {
		t.Fatalf("EncryptToFile: %v", err)
	}

	decrypted, err := DecryptFile(notePath, identity)
	if err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted file content mismatch")
	}
}

func TestPublicKeyExport(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.txt")

	identity, err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	pubKey := PublicKey(identity)
	if pubKey == "" {
		t.Error("public key is empty")
	}
	if len(pubKey) < 50 {
		t.Errorf("public key seems too short: %s", pubKey)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/crypto/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write crypto.go implementation**

```go
package crypto

import (
	"bytes"
	"io"
	"os"

	"filippo.io/age"
)

func GenerateKey(keyPath string) (*age.HybridIdentity, error) {
	identity, err := age.GenerateHybridIdentity()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyPath, []byte(identity.String()+"\n"), 0600); err != nil {
		return nil, err
	}

	return identity, nil
}

func LoadIdentity(keyPath string) (*age.HybridIdentity, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	identities, err := age.ParseIdentities(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	for _, id := range identities {
		if hybrid, ok := id.(*age.HybridIdentity); ok {
			return hybrid, nil
		}
	}

	return nil, os.ErrNotExist
}

func PublicKey(identity *age.HybridIdentity) string {
	return identity.Recipient().String()
}

func Encrypt(plaintext []byte, recipients ...age.Recipient) ([]byte, error) {
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipients...)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decrypt(ciphertext []byte, identities ...age.Identity) ([]byte, error) {
	r, err := age.Decrypt(bytes.NewReader(ciphertext), identities...)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func EncryptToFile(path string, plaintext []byte, recipients ...age.Recipient) error {
	encrypted, err := Encrypt(plaintext, recipients...)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encrypted, 0600)
}

func DecryptFile(path string, identities ...age.Identity) ([]byte, error) {
	ciphertext, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Decrypt(ciphertext, identities...)
}
```

- [ ] **Step 4: Add age dependency and run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go get filippo.io/age@latest
go test ./internal/crypto/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/crypto/ go.mod go.sum
git commit -m "feat: add crypto package with age post-quantum key generation and encryption"
```

---

### Task 5: Note Model & Frontmatter Parsing

**Files:**
- Create: `internal/notes/note.go`
- Create: `internal/notes/note_test.go`

- [ ] **Step 1: Write failing tests for note model and frontmatter**

```go
package notes

import (
	"strings"
	"testing"
	"time"
)

func TestParseNote(t *testing.T) {
	content := `---
customer: Acme Corp
type: meeting
created: 06-05-2026 09:30
due: 10-05-2026 17:00
tags: [meeting, followup]
status: open
attendees: [Kim, Sarah]
---
# Kickoff Meeting

## Agenda
- Discuss timeline
`
	note, err := ParseNote(content, "eu")
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if note.Customer != "Acme Corp" {
		t.Errorf("customer: expected Acme Corp, got %s", note.Customer)
	}
	if note.Type != Meeting {
		t.Errorf("type: expected meeting, got %s", note.Type)
	}
	if note.Status != StatusOpen {
		t.Errorf("status: expected open, got %s", note.Status)
	}
	if len(note.Tags) != 2 {
		t.Errorf("tags: expected 2, got %d", len(note.Tags))
	}
	if len(note.Attendees) != 2 {
		t.Errorf("attendees: expected 2, got %d", len(note.Attendees))
	}
	if !strings.Contains(note.Body, "## Agenda") {
		t.Error("body should contain Agenda section")
	}
}

func TestGenerateTemplate(t *testing.T) {
	note := &Note{
		Customer: "Red Hat",
		Type:     Task,
		Created:  time.Date(2026, 5, 6, 9, 30, 0, 0, time.Local),
		Due:      time.Date(2026, 5, 10, 17, 0, 0, 0, time.Local),
		Tags:     []string{"sprint"},
		Status:   StatusOpen,
		Priority: PriorityHigh,
		Title:    "Sprint Review",
	}

	content := GenerateTemplate(note, "eu")
	if !strings.Contains(content, "customer: Red Hat") {
		t.Error("template should contain customer")
	}
	if !strings.Contains(content, "type: task") {
		t.Error("template should contain type")
	}
	if !strings.Contains(content, "priority: high") {
		t.Error("template should contain priority")
	}
	if !strings.Contains(content, "# Sprint Review") {
		t.Error("template should contain title")
	}
	if !strings.Contains(content, "## Description") {
		t.Error("task template should contain Description section")
	}
}

func TestSanitizeCustomerName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Acme Corp", "Acme-Corp"},
		{"Red Hat", "Red-Hat"},
		{"my company!@#", "my-company"},
		{"  spaces  ", "spaces"},
	}
	for _, tt := range tests {
		got := SanitizeCustomerName(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeCustomerName(%q): expected %q, got %q", tt.input, tt.expected, got)
		}
	}
}

func TestNoteFilename(t *testing.T) {
	created := time.Date(2026, 5, 6, 9, 30, 0, 0, time.Local)
	got := NoteFilename("Kickoff Meeting", created)
	expected := "2026-05-06-kickoff-meeting.md.age"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestNoteTypes(t *testing.T) {
	types := []NoteType{Meeting, Task, Reminder, Followup}
	for _, nt := range types {
		if nt.String() == "" {
			t.Errorf("NoteType %d has empty string", nt)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/notes/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write note.go implementation**

```go
package notes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type NoteType string

const (
	Meeting  NoteType = "meeting"
	Task     NoteType = "task"
	Reminder NoteType = "reminder"
	Followup NoteType = "followup"
)

func (n NoteType) String() string { return string(n) }

type Status string

const (
	StatusOpen Status = "open"
	StatusDone Status = "done"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

type Note struct {
	Customer  string   `yaml:"customer"`
	Type      NoteType `yaml:"type"`
	Created   time.Time `yaml:"-"`
	CreatedRaw string  `yaml:"created"`
	Due       time.Time `yaml:"-"`
	DueRaw    string   `yaml:"due,omitempty"`
	Repeat    string   `yaml:"repeat,omitempty"`
	Tags      []string `yaml:"tags"`
	Status    Status   `yaml:"status"`
	Priority  Priority `yaml:"priority,omitempty"`
	Attendees []string `yaml:"attendees,omitempty"`
	Related   string   `yaml:"related,omitempty"`
	Title     string   `yaml:"-"`
	Body      string   `yaml:"-"`
	FilePath  string   `yaml:"-"`
}

func ParseNote(content, dateFormat string) (*Note, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid frontmatter")
	}

	var note Note
	if err := yaml.Unmarshal([]byte(parts[1]), &note); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	note.Body = strings.TrimSpace(parts[2])

	if note.CreatedRaw != "" {
		if t, err := parseFrontmatterDate(note.CreatedRaw, dateFormat); err == nil {
			note.Created = t
		}
	}
	if note.DueRaw != "" {
		if t, err := parseFrontmatterDate(note.DueRaw, dateFormat); err == nil {
			note.Due = t
		}
	}

	lines := strings.SplitN(note.Body, "\n", 2)
	if len(lines) > 0 {
		note.Title = strings.TrimPrefix(strings.TrimSpace(lines[0]), "# ")
	}

	return &note, nil
}

func parseFrontmatterDate(s, format string) (time.Time, error) {
	s = strings.TrimSpace(s)
	var layouts []string
	if format == "us" {
		layouts = []string{"01-02-2006 15:04", "01-02-2006"}
	} else {
		layouts = []string{"02-01-2006 15:04", "02-01-2006"}
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %q", s)
}

func GenerateTemplate(note *Note, dateFormat string) string {
	fm := &frontmatterOut{
		Customer:  note.Customer,
		Type:      string(note.Type),
		Created:   formatFrontmatterDate(note.Created, dateFormat),
		Due:       formatFrontmatterDate(note.Due, dateFormat),
		Repeat:    note.Repeat,
		Tags:      note.Tags,
		Status:    string(note.Status),
		Priority:  string(note.Priority),
		Attendees: note.Attendees,
		Related:   note.Related,
	}

	data, _ := yaml.Marshal(fm)
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(data)
	sb.WriteString("---\n")
	sb.WriteString("# " + note.Title + "\n")

	switch note.Type {
	case Meeting:
		sb.WriteString("\n## Agenda\n- \n\n## Notes\n- \n\n## Action Items\n- [ ] \n")
	case Task:
		sb.WriteString("\n## Description\n\n## Acceptance Criteria\n- [ ] \n\n## Notes\n")
	case Reminder:
		sb.WriteString("\n")
	case Followup:
		sb.WriteString("\n## What was agreed\n\n## What needs to happen\n- [ ] \n\n## Status update\n")
	}

	return sb.String()
}

type frontmatterOut struct {
	Customer  string   `yaml:"customer"`
	Type      string   `yaml:"type"`
	Created   string   `yaml:"created"`
	Due       string   `yaml:"due,omitempty"`
	Repeat    string   `yaml:"repeat,omitempty"`
	Tags      []string `yaml:"tags"`
	Status    string   `yaml:"status"`
	Priority  string   `yaml:"priority,omitempty"`
	Attendees []string `yaml:"attendees,omitempty"`
	Related   string   `yaml:"related,omitempty"`
}

func formatFrontmatterDate(t time.Time, format string) string {
	if t.IsZero() {
		return ""
	}
	if t.Hour() == 0 && t.Minute() == 0 {
		if format == "us" {
			return t.Format("01-02-2006")
		}
		return t.Format("02-01-2006")
	}
	if format == "us" {
		return t.Format("01-02-2006 15:04")
	}
	return t.Format("02-01-2006 15:04")
}

var sanitizeRe = regexp.MustCompile(`[^a-zA-Z0-9-]`)

func SanitizeCustomerName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = sanitizeRe.ReplaceAllString(name, "")
	name = strings.Trim(name, "-")
	return name
}

func NoteFilename(title string, created time.Time) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = sanitizeRe.ReplaceAllString(slug, "")
	slug = strings.Trim(slug, "-")
	return created.Format("2006-01-02") + "-" + slug + ".md.age"
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/notes/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/notes/note.go internal/notes/note_test.go
git commit -m "feat: add note model with frontmatter parsing and template generation"
```

---

### Task 6: Note Store (CRUD with Encryption)

**Files:**
- Create: `internal/notes/store.go`
- Modify: `internal/notes/store_test.go` (add to existing test file, or create new)

- [ ] **Step 1: Write failing tests for NoteStore**

Create `internal/notes/store_test.go`:

```go
package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

func setupTestStore(t *testing.T) (*NoteStore, string) {
	t.Helper()
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.txt")
	identity, err := crypto.GenerateKey(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	store := NewNoteStore(tmpDir, identity, "eu")
	return store, tmpDir
}

func TestStoreCreate(t *testing.T) {
	store, tmpDir := setupTestStore(t)

	note := &Note{
		Customer: "Acme Corp",
		Type:     Meeting,
		Created:  time.Date(2026, 5, 6, 9, 30, 0, 0, time.Local),
		Title:    "Kickoff Meeting",
		Tags:     []string{"meeting"},
		Status:   StatusOpen,
	}

	path, err := store.Create(note)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	expectedDir := filepath.Join(tmpDir, "Acme-Corp")
	if _, err := os.Stat(expectedDir); err != nil {
		t.Errorf("customer dir not created: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("note file not created: %v", err)
	}
}

func TestStoreListAndRead(t *testing.T) {
	store, _ := setupTestStore(t)

	note := &Note{
		Customer: "Red Hat",
		Type:     Task,
		Created:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.Local),
		Due:      time.Date(2026, 5, 10, 17, 0, 0, 0, time.Local),
		Title:    "Sprint Review",
		Tags:     []string{"sprint"},
		Status:   StatusOpen,
		Priority: PriorityHigh,
	}

	path, err := store.Create(note)
	if err != nil {
		t.Fatal(err)
	}

	notes, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}

	readNote, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if readNote.Customer != "Red Hat" {
		t.Errorf("expected customer Red Hat, got %s", readNote.Customer)
	}
	if readNote.Priority != PriorityHigh {
		t.Errorf("expected priority high, got %s", readNote.Priority)
	}
}

func TestStoreUpdate(t *testing.T) {
	store, _ := setupTestStore(t)

	note := &Note{
		Customer: "Test Co",
		Type:     Reminder,
		Created:  time.Now(),
		Title:    "Test Reminder",
		Status:   StatusOpen,
	}

	path, err := store.Create(note)
	if err != nil {
		t.Fatal(err)
	}

	readNote, err := store.Read(path)
	if err != nil {
		t.Fatal(err)
	}

	readNote.Status = StatusDone
	if err := store.Update(path, readNote); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := store.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusDone {
		t.Errorf("expected status done, got %s", updated.Status)
	}
}

func TestStoreSearch(t *testing.T) {
	store, _ := setupTestStore(t)

	notes := []*Note{
		{Customer: "Acme", Type: Meeting, Created: time.Now(), Title: "API Review", Tags: []string{"api"}, Status: StatusOpen},
		{Customer: "Acme", Type: Task, Created: time.Now(), Title: "Fix Bug", Tags: []string{"bug"}, Status: StatusOpen},
		{Customer: "Other", Type: Reminder, Created: time.Now(), Title: "Call Bob", Status: StatusOpen},
	}
	for _, n := range notes {
		if _, err := store.Create(n); err != nil {
			t.Fatal(err)
		}
	}

	results, err := store.Search("API")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'API', got %d", len(results))
	}

	results, err = store.Search("Acme")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'Acme', got %d", len(results))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/notes/ -v -run TestStore
```
Expected: FAIL — NoteStore not defined

- [ ] **Step 3: Write store.go implementation**

```go
package notes

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"filippo.io/age"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

type NoteStore struct {
	baseDir    string
	identity   *age.HybridIdentity
	dateFormat string
}

func NewNoteStore(baseDir string, identity *age.HybridIdentity, dateFormat string) *NoteStore {
	return &NoteStore{
		baseDir:    baseDir,
		identity:   identity,
		dateFormat: dateFormat,
	}
}

func (s *NoteStore) Create(note *Note) (string, error) {
	customerDir := filepath.Join(s.baseDir, SanitizeCustomerName(note.Customer))
	if err := os.MkdirAll(customerDir, 0700); err != nil {
		return "", err
	}

	filename := NoteFilename(note.Title, note.Created)
	notePath := filepath.Join(customerDir, filename)

	content := GenerateTemplate(note, s.dateFormat)

	if err := crypto.EncryptToFile(notePath, []byte(content), s.identity.Recipient()); err != nil {
		return "", err
	}

	return notePath, nil
}

func (s *NoteStore) Read(path string) (*Note, error) {
	plaintext, err := crypto.DecryptFile(path, s.identity)
	if err != nil {
		return nil, err
	}
	note, err := ParseNote(string(plaintext), s.dateFormat)
	if err != nil {
		return nil, err
	}
	note.FilePath = path
	return note, nil
}

func (s *NoteStore) Update(path string, note *Note) error {
	content := GenerateTemplate(note, s.dateFormat)
	if note.Body != "" && !bodyMatchesTemplate(note) {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) == 3 {
			content = "---" + parts[1] + "---\n" + note.Body
		}
	}
	return crypto.EncryptToFile(path, []byte(content), s.identity.Recipient())
}

func bodyMatchesTemplate(note *Note) bool {
	return note.Body == ""
}

func (s *NoteStore) List() ([]*Note, error) {
	var notes []*Note

	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".pq-notes" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md.age") {
			return nil
		}
		note, err := s.Read(path)
		if err != nil {
			return nil
		}
		notes = append(notes, note)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Created.After(notes[j].Created)
	})

	return notes, nil
}

func (s *NoteStore) Search(query string) ([]*Note, error) {
	allNotes, err := s.List()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []*Note
	for _, note := range allNotes {
		if matchesSearch(note, query) {
			results = append(results, note)
		}
	}
	return results, nil
}

func matchesSearch(note *Note, query string) bool {
	if strings.Contains(strings.ToLower(note.Title), query) {
		return true
	}
	if strings.Contains(strings.ToLower(note.Customer), query) {
		return true
	}
	if strings.Contains(strings.ToLower(note.Body), query) {
		return true
	}
	for _, tag := range note.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/notes/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/notes/store.go internal/notes/store_test.go
git commit -m "feat: add NoteStore with encrypted CRUD and search"
```

---

### Task 7: Calendar & Working Days Package

**Files:**
- Create: `internal/calendar/calendar.go`
- Create: `internal/calendar/calendar_test.go`

- [ ] **Step 1: Write failing tests for business calendar**

```go
package calendar

import (
	"testing"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/config"
)

func TestIsWorkday(t *testing.T) {
	cfg := &config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
	}
	cal := New(cfg)

	// Wednesday May 6, 2026
	wed := time.Date(2026, 5, 6, 12, 0, 0, 0, time.Local)
	if !cal.IsWorkday(wed) {
		t.Error("Wednesday should be a workday")
	}

	// Saturday May 9, 2026
	sat := time.Date(2026, 5, 9, 12, 0, 0, 0, time.Local)
	if cal.IsWorkday(sat) {
		t.Error("Saturday should not be a workday")
	}

	// Sunday May 10, 2026
	sun := time.Date(2026, 5, 10, 12, 0, 0, 0, time.Local)
	if cal.IsWorkday(sun) {
		t.Error("Sunday should not be a workday")
	}
}

func TestIsWorkdaySaudiArabia(t *testing.T) {
	cfg := &config.Config{
		Country: "SA",
		Weekend: []string{"friday", "saturday"},
	}
	cal := New(cfg)

	// Friday May 8, 2026
	fri := time.Date(2026, 5, 8, 12, 0, 0, 0, time.Local)
	if cal.IsWorkday(fri) {
		t.Error("Friday should not be a workday in SA")
	}

	// Sunday May 10, 2026
	sun := time.Date(2026, 5, 10, 12, 0, 0, 0, time.Local)
	if !cal.IsWorkday(sun) {
		t.Error("Sunday should be a workday in SA")
	}
}

func TestNthLastWorkday(t *testing.T) {
	cfg := &config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
	}
	cal := New(cfg)

	// Last workday of May 2026
	last := cal.NthLastWorkday(2026, time.May, 1)
	if last.Day() != 29 {
		t.Errorf("last workday of May 2026: expected 29, got %d", last.Day())
	}

	// 2nd to last workday of May 2026
	secondLast := cal.NthLastWorkday(2026, time.May, 2)
	if secondLast.Day() != 28 {
		t.Errorf("2nd-last workday of May 2026: expected 28, got %d", secondLast.Day())
	}
}

func TestCustomHoliday(t *testing.T) {
	cfg := &config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
		CustomHolidays: []config.CustomHoliday{
			{Name: "Company Day", Date: "15-06"},
		},
	}
	cal := New(cfg)

	// June 15, 2026 is a Monday — should not be a workday due to custom holiday
	companyDay := time.Date(2026, 6, 15, 12, 0, 0, 0, time.Local)
	if cal.IsWorkday(companyDay) {
		t.Error("Company Day (June 15) should not be a workday")
	}
}

func TestWorkdaysFrom(t *testing.T) {
	cfg := &config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
	}
	cal := New(cfg)

	// 5 workdays from Wednesday May 6
	start := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	result := cal.WorkdaysFrom(start, 5)
	// May 6(W)+7(Th)+8(F)+11(M)+12(Tu)+13(W) — 5 workdays ahead = May 13
	if result.Day() != 13 {
		t.Errorf("5 workdays from May 6: expected May 13, got %v", result)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/calendar/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write calendar.go implementation**

```go
package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/aa"
	"github.com/rickar/cal/v2/at"
	"github.com/rickar/cal/v2/au"
	"github.com/rickar/cal/v2/be"
	"github.com/rickar/cal/v2/ca"
	"github.com/rickar/cal/v2/ch"
	"github.com/rickar/cal/v2/cz"
	"github.com/rickar/cal/v2/de"
	"github.com/rickar/cal/v2/dk"
	"github.com/rickar/cal/v2/es"
	"github.com/rickar/cal/v2/fi"
	"github.com/rickar/cal/v2/fr"
	"github.com/rickar/cal/v2/gb"
	"github.com/rickar/cal/v2/ie"
	"github.com/rickar/cal/v2/it"
	"github.com/rickar/cal/v2/nl"
	"github.com/rickar/cal/v2/no"
	"github.com/rickar/cal/v2/nz"
	"github.com/rickar/cal/v2/pl"
	"github.com/rickar/cal/v2/pt"
	"github.com/rickar/cal/v2/se"
	"github.com/rickar/cal/v2/us"
)

type BusinessCal struct {
	bc *cal.BusinessCalendar
}

func New(cfg *config.Config) *BusinessCal {
	bc := cal.NewBusinessCalendar()

	for _, day := range []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday,
		time.Thursday, time.Friday, time.Saturday, time.Sunday,
	} {
		bc.SetWorkday(day, !isWeekend(day, cfg.Weekend))
	}

	addCountryHolidays(bc, cfg.Country)
	addCustomHolidays(bc, cfg.CustomHolidays)

	return &BusinessCal{bc: bc}
}

func (c *BusinessCal) IsWorkday(t time.Time) bool {
	return c.bc.IsWorkday(t)
}

func (c *BusinessCal) WorkdaysFrom(start time.Time, offset int) time.Time {
	return c.bc.WorkdaysFrom(start, offset)
}

func (c *BusinessCal) NthLastWorkday(year int, month time.Month, n int) time.Time {
	end := cal.MonthEnd(time.Date(year, month, 1, 0, 0, 0, 0, time.Local))
	count := 0
	for d := end; d.Month() == month; d = d.AddDate(0, 0, -1) {
		if c.bc.IsWorkday(d) {
			count++
			if count == n {
				return d
			}
		}
	}
	return time.Time{}
}

func isWeekend(day time.Weekday, weekendDays []string) bool {
	dayName := strings.ToLower(day.String())
	for _, wd := range weekendDays {
		if strings.ToLower(wd) == dayName {
			return true
		}
	}
	return false
}

func addCountryHolidays(bc *cal.BusinessCalendar, country string) {
	holidays := countryHolidays(strings.ToUpper(country))
	bc.AddHoliday(holidays...)
}

func countryHolidays(country string) []*cal.Holiday {
	switch country {
	case "AT":
		return []*cal.Holiday{at.Neujahr, at.HeiligeDreiKoenige, at.Ostermontag, at.TagDerArbeit, at.ChristiHimmelfahrt, at.Pfingstmontag, at.Fronleichnam, at.MariaHimmelfahrt, at.Nationalfeiertag, at.Allerheiligen, at.MariaEmpfaengnis, at.Christtag, at.Stefanitag}
	case "AU":
		return []*cal.Holiday{au.NewYear, au.AustraliaDay, au.GoodFriday, au.EasterSaturday, au.EasterMonday, au.AnzacDay, au.QueensBirthday, au.ChristmasDay, au.BoxingDay}
	case "BE":
		return []*cal.Holiday{be.NieuwJaar, be.Paasmaandag, be.DagVanDeArbeid, be.OnzeLieveHeerHemelvaart, be.Pinkstermaandag, be.NationaleFeestdag, be.OnzeLieveVrouwHemelvaart, be.Allerheiligen, be.Wapenstilstand, be.Kerstmis}
	case "CA":
		return []*cal.Holiday{ca.NewYear, ca.GoodFriday, ca.CanadaDay, ca.LabourDay, ca.ThanksgivingDay, ca.ChristmasDay, ca.BoxingDay}
	case "CH":
		return []*cal.Holiday{ch.Neujahrstag, ch.Berchtoldstag, ch.Karfreitag, ch.Ostermontag, ch.TagDerArbeit, ch.AuffahrtChristiHimmelfahrt, ch.Pfingstmontag, ch.Bundesfeiertag, ch.Weihnachtstag, ch.Stephanstag}
	case "CZ":
		return []*cal.Holiday{cz.DenObnovySamostatnostiCeskehoStatu, cz.VelkyPatek, cz.VelikonocniPondeli, cz.SvatekPrace, cz.DenVitezstvi, cz.DenSlovanskychVerozvest, cz.DenUpaleninMistraJanaHusa, cz.DenCeskeStatnosti, cz.DenVznikuSamostatnehoStatu, cz.StedryDen, cz.PrvniSvatekVanocni, cz.DruhySvatekVanocni}
	case "DE":
		return []*cal.Holiday{de.Neujahr, de.Karfreitag, de.Ostermontag, de.TagDerArbeit, de.ChristiHimmelfahrt, de.Pfingstmontag, de.TagDerDeutschenEinheit, de.Weihnachtstag, de.ZweiterWeihnachtstag}
	case "DK":
		return []*cal.Holiday{dk.Nytaarsdag, dk.Skaertorsdag, dk.Langfredag, dk.AndenPaaskedag, dk.StoreBededag, dk.KristiHimmelfartsdag, dk.AndenPinsedag, dk.Juledag, dk.AndenJuledag}
	case "ES":
		return []*cal.Holiday{es.AnoNuevo, es.Epifania, es.ViernesSanto, es.DiaDeLaAsuncion, es.FiestaDelTrabajo, es.FiestaNacional, es.TodosLosSantos, es.DiaDelaConstitucion, es.InmaculadaConcepcion, es.Navidad}
	case "FI":
		return []*cal.Holiday{fi.Uudenvuodenpaiva, fi.Loppiainen, fi.Pitkaperiantai, fi.Paasiaispaiva, fi.Toisen, fi.Vappu, fi.HelatorstaiChristiHimmelfahrt, fi.Juhannuspaiva, fi.Pyhainpaiva, fi.Itsenaisyyspaiva, fi.Joulupaiva, fi.Tapaninpaiva}
	case "FR":
		return []*cal.Holiday{fr.NouvelAn, fr.LundiDePaques, fr.FeteDuTravail, fr.Armistice1945, fr.Ascension, fr.LundiDePentecote, fr.FeteNationale, fr.Assomption, fr.Toussaint, fr.Armistice1918, fr.Noel}
	case "GB":
		return []*cal.Holiday{gb.NewYear, gb.GoodFriday, gb.EasterMonday, gb.EarlyMay, gb.SpringHoliday, gb.SummerHoliday, gb.ChristmasDay, gb.BoxingDay}
	case "IE":
		return []*cal.Holiday{ie.NewYear, ie.StPatricksDay, ie.EasterMonday, ie.MayHoliday, ie.JuneHoliday, ie.AugustHoliday, ie.OctoberHoliday, ie.ChristmasDay, ie.StStephensDay}
	case "IT":
		return []*cal.Holiday{it.Capodanno, it.Epifania, it.Pasquetta, it.FestaDellaLiberazione, it.FestaDeiLavoratori, it.FestaDellaRepubblica, it.Ferragosto, it.TuttiISanti, it.ImmacolataConcezione, it.Natale, it.SantoStefano}
	case "NL":
		return []*cal.Holiday{nl.Nieuwjaarsdag, nl.TweedesPaasdag, nl.Koningsdag, nl.Bevrijdingsdag, nl.Hemelvaartsdag, nl.TweedePinksterdag, nl.EersteKerstdag, nl.TweedeKerstdag}
	case "NO":
		return []*cal.Holiday{no.Nyttarsdag, no.Skjaertorsdag, no.Langfredag, no.FoerstePaaskedag, no.AndrePaaskedag, no.Arbeiderenesdag, no.Grunnlovsdag, no.KristiHimmelfartsdag, no.FoerstePinsedag, no.AndrePinsedag, no.FoersteJuledag, no.AndreJuledag}
	case "NZ":
		return []*cal.Holiday{nz.NewYear, nz.DayAfterNewYear, nz.WaitangiDay, nz.GoodFriday, nz.EasterMonday, nz.AnzacDay, nz.QueensBirthday, nz.LabourDay, nz.ChristmasDay, nz.BoxingDay}
	case "PL":
		return []*cal.Holiday{pl.NowyRok, pl.SwietoTrzechKroli, pl.PoniedzialekWielkanocny, pl.SwietoPracy, pl.SwietoKonstytucji3Maja, pl.ZeslanieDuchaSwietego, pl.BozeCialo, pl.WniebowziecieNMP, pl.WszystkichSwietych, pl.SwietoNiepodleglosci, pl.BozemNarodzenie, pl.DrugiBozemNarodzenie}
	case "PT":
		return []*cal.Holiday{pt.AnoNovo, pt.SextaFeiraSanta, pt.DiaDaLiberdade, pt.DiasDoTrabalhador, pt.CorpusChristi, pt.DiaDePortugal, pt.AssuncaoDeNossaSenhora, pt.ImplantacaoDaRepublica, pt.DiaDeTodosOsSantos, pt.RestauracaoDaIndependencia, pt.ImaculadaConceicao, pt.Natal}
	case "SE":
		return []*cal.Holiday{se.Nyarsdagen, se.TrettondedagJul, se.Langfredagen, se.Paskdagen, se.AnnandagPask, se.ForstaMaj, se.KristiHimmelsfardsdag, se.Pingstdagen, se.Nationaldagen, se.Midsommardagen, se.AllaHelgonsDag, se.Juldagen, se.AnnandagJul}
	case "US":
		return []*cal.Holiday{us.NewYear, us.MLKDay, us.PresidentsDay, us.MemorialDay, us.IndependenceDay, us.LaborDay, us.ColumbusDay, us.VeteransDay, us.ThanksgivingDay, us.ChristmasDay}
	default:
		return []*cal.Holiday{aa.NewYear, aa.GoodFriday, aa.EasterMonday, aa.ChristmasDay, aa.ChristmasDay2}
	}
}

func addCustomHolidays(bc *cal.BusinessCalendar, holidays []config.CustomHoliday) {
	for _, ch := range holidays {
		parts := strings.Split(ch.Date, "-")
		if len(parts) != 2 {
			continue
		}
		var day, month int
		fmt.Sscanf(parts[0], "%d", &day)
		fmt.Sscanf(parts[1], "%d", &month)

		h := &cal.Holiday{
			Name:  ch.Name,
			Type:  cal.ObservanceOther,
			Month: time.Month(month),
			Day:   day,
			Func:  cal.CalcDayOfMonth,
		}
		bc.AddHoliday(h)
	}
}
```

- [ ] **Step 4: Add rickar/cal dependency and run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go get github.com/rickar/cal/v2@latest
go test ./internal/calendar/ -v
```
Expected: all tests PASS

Note: Some country package imports may need adjustment if specific holiday variables have different names. Fix any compilation errors by checking the actual exported names in the `rickar/cal` subpackages. The test itself validates the business logic (workday/weekend/custom holiday detection), not the specific holiday variable names. If a country subpackage doesn't exist or has different exports, remove or adjust that case in the switch statement and re-run.

- [ ] **Step 5: Commit**

```bash
git add internal/calendar/ go.mod go.sum
git commit -m "feat: add calendar package with country holidays and configurable weekends"
```

---

### Task 8: Repeating Schedule Parser

**Files:**
- Create: `internal/schedule/schedule.go`
- Create: `internal/schedule/schedule_test.go`

- [ ] **Step 1: Write failing tests for schedule parsing**

```go
package schedule

import (
	"testing"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/calendar"
	"github.com/kborup-redhat/pq-notes/internal/config"
)

func testCal() *calendar.BusinessCal {
	return calendar.New(&config.Config{
		Country: "DK",
		Weekend: []string{"saturday", "sunday"},
	})
}

func TestParseRepeat(t *testing.T) {
	tests := []struct {
		input string
		kind  RepeatKind
	}{
		{"daily", Daily},
		{"weekly", Weekly},
		{"monthly 15", MonthlyDay},
		{"every monday", WeeklyDay},
		{"every 1st", MonthlyDay},
		{"every last workday", LastWorkday},
		{"every 2nd-last workday", NthLastWorkday},
	}
	for _, tt := range tests {
		r, err := ParseRepeat(tt.input)
		if err != nil {
			t.Errorf("ParseRepeat(%q): %v", tt.input, err)
			continue
		}
		if r.Kind != tt.kind {
			t.Errorf("ParseRepeat(%q): expected kind %v, got %v", tt.input, tt.kind, r.Kind)
		}
	}
}

func TestNextOccurrenceDaily(t *testing.T) {
	r, _ := ParseRepeat("daily")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 7, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceWeekly(t *testing.T) {
	r, _ := ParseRepeat("weekly")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local) // Wednesday
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 13, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceEveryMonday(t *testing.T) {
	r, _ := ParseRepeat("every monday")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local) // Wednesday
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceLastWorkday(t *testing.T) {
	r, _ := ParseRepeat("every last workday")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	// Last workday of May 2026 is May 29 (Friday)
	if next.Day() != 29 || next.Month() != 5 {
		t.Errorf("expected May 29, got %v", next)
	}
}

func TestNextOccurrence2ndLastWorkday(t *testing.T) {
	r, _ := ParseRepeat("every 2nd-last workday")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	// 2nd-last workday of May 2026 is May 28 (Thursday)
	if next.Day() != 28 || next.Month() != 5 {
		t.Errorf("expected May 28, got %v", next)
	}
}

func TestNextOccurrenceMonthly15(t *testing.T) {
	r, _ := ParseRepeat("monthly 15")
	cal := testCal()
	from := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 5, 15, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestNextOccurrenceMonthly15AfterThe15th(t *testing.T) {
	r, _ := ParseRepeat("monthly 15")
	cal := testCal()
	from := time.Date(2026, 5, 20, 0, 0, 0, 0, time.Local)
	next := r.NextOccurrence(from, cal)
	expected := time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/schedule/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write schedule.go implementation**

```go
package schedule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kborup-redhat/pq-notes/internal/calendar"
)

type RepeatKind int

const (
	Daily RepeatKind = iota
	Weekly
	WeeklyDay
	BiWeeklyDay
	MonthlyDay
	LastWorkday
	NthLastWorkday
)

type Repeat struct {
	Kind    RepeatKind
	Weekday time.Weekday
	Day     int
	N       int
	Raw     string
}

var nthLastRe = regexp.MustCompile(`every\s+(\d+)(?:st|nd|rd|th)-last\s+workday`)

func ParseRepeat(input string) (*Repeat, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "daily":
		return &Repeat{Kind: Daily, Raw: input}, nil
	case "weekly":
		return &Repeat{Kind: Weekly, Raw: input}, nil
	case "every last workday":
		return &Repeat{Kind: LastWorkday, N: 1, Raw: input}, nil
	}

	if m := nthLastRe.FindStringSubmatch(input); m != nil {
		n, _ := strconv.Atoi(m[1])
		return &Repeat{Kind: NthLastWorkday, N: n, Raw: input}, nil
	}

	weekdays := map[string]time.Weekday{
		"monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
		"thursday": time.Thursday, "friday": time.Friday,
		"saturday": time.Saturday, "sunday": time.Sunday,
	}

	if strings.HasPrefix(input, "every ") {
		dayName := strings.TrimPrefix(input, "every ")
		if wd, ok := weekdays[dayName]; ok {
			return &Repeat{Kind: WeeklyDay, Weekday: wd, Raw: input}, nil
		}

		if strings.HasSuffix(dayName, "st") || strings.HasSuffix(dayName, "nd") || strings.HasSuffix(dayName, "rd") || strings.HasSuffix(dayName, "th") {
			numStr := dayName[:len(dayName)-2]
			if d, err := strconv.Atoi(numStr); err == nil {
				return &Repeat{Kind: MonthlyDay, Day: d, Raw: input}, nil
			}
		}
	}

	if strings.HasPrefix(input, "monthly ") {
		dayStr := strings.TrimPrefix(input, "monthly ")
		if d, err := strconv.Atoi(dayStr); err == nil {
			return &Repeat{Kind: MonthlyDay, Day: d, Raw: input}, nil
		}
	}

	biweeklyRe := regexp.MustCompile(`every\s+2\s+weeks\s+(\w+)`)
	if m := biweeklyRe.FindStringSubmatch(input); m != nil {
		if wd, ok := weekdays[m[1]]; ok {
			return &Repeat{Kind: BiWeeklyDay, Weekday: wd, Raw: input}, nil
		}
	}

	return nil, fmt.Errorf("unrecognized repeat pattern: %q", input)
}

func (r *Repeat) NextOccurrence(from time.Time, cal *calendar.BusinessCal) time.Time {
	switch r.Kind {
	case Daily:
		return from.AddDate(0, 0, 1)

	case Weekly:
		return from.AddDate(0, 0, 7)

	case WeeklyDay:
		return nextWeekday(from, r.Weekday)

	case BiWeeklyDay:
		next := nextWeekday(from, r.Weekday)
		return next.AddDate(0, 0, 7)

	case MonthlyDay:
		candidate := time.Date(from.Year(), from.Month(), r.Day, 0, 0, 0, 0, from.Location())
		if !candidate.After(from) {
			candidate = candidate.AddDate(0, 1, 0)
		}
		return candidate

	case LastWorkday:
		return cal.NthLastWorkday(from.Year(), from.Month(), 1)

	case NthLastWorkday:
		candidate := cal.NthLastWorkday(from.Year(), from.Month(), r.N)
		if !candidate.After(from) {
			nextMonth := from.AddDate(0, 1, 0)
			candidate = cal.NthLastWorkday(nextMonth.Year(), nextMonth.Month(), r.N)
		}
		return candidate
	}

	return time.Time{}
}

func nextWeekday(from time.Time, target time.Weekday) time.Time {
	daysAhead := int(target) - int(from.Weekday())
	if daysAhead <= 0 {
		daysAhead += 7
	}
	next := from.AddDate(0, 0, daysAhead)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, from.Location())
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/schedule/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/schedule/
git commit -m "feat: add schedule parser for repeating notes (daily, weekly, nth-last workday, etc.)"
```

---

### Task 9: Editor Integration

**Files:**
- Create: `internal/editor/editor.go`
- Create: `internal/editor/editor_test.go`

- [ ] **Step 1: Write failing test for editor command building**

```go
package editor

import (
	"testing"
)

func TestEditorCommand(t *testing.T) {
	tests := []struct {
		editor   string
		filePath string
		wantCmd  string
		wantArgs []string
	}{
		{"vi", "/tmp/test.md", "vi", []string{"/tmp/test.md"}},
		{"code", "/tmp/test.md", "code", []string{"--wait", "/tmp/test.md"}},
	}
	for _, tt := range tests {
		cmd, args := BuildCommand(tt.editor, tt.filePath)
		if cmd != tt.wantCmd {
			t.Errorf("editor=%s: expected cmd %s, got %s", tt.editor, tt.wantCmd, cmd)
		}
		if len(args) != len(tt.wantArgs) {
			t.Errorf("editor=%s: expected %d args, got %d", tt.editor, len(tt.wantArgs), len(args))
			continue
		}
		for i := range args {
			if args[i] != tt.wantArgs[i] {
				t.Errorf("editor=%s: arg %d: expected %s, got %s", tt.editor, i, tt.wantArgs[i], args[i])
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/editor/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write editor.go implementation**

```go
package editor

import (
	"os"
	"os/exec"
)

func BuildCommand(editor, filePath string) (string, []string) {
	switch editor {
	case "code":
		return "code", []string{"--wait", filePath}
	default:
		return editor, []string{filePath}
	}
}

func Open(editor, filePath string) error {
	cmdName, args := BuildCommand(editor, filePath)
	cmd := exec.Command(cmdName, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

- [ ] **Step 4: Run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/editor/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/editor/
git commit -m "feat: add editor package with vi and VS Code support"
```

---

### Task 10: Contact Management

**Files:**
- Create: `internal/share/contacts.go`
- Create: `internal/share/share.go`
- Create: `internal/share/share_test.go`

- [ ] **Step 1: Write failing tests for contacts and sharing**

```go
package share

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

func TestContactsCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	contactsFile := filepath.Join(tmpDir, "contacts.yaml")

	if err := AddContact(contactsFile, "Alice", "age1pq1testkey123"); err != nil {
		t.Fatalf("AddContact: %v", err)
	}

	contacts, err := LoadContacts(contactsFile)
	if err != nil {
		t.Fatalf("LoadContacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].Name != "Alice" {
		t.Errorf("expected name Alice, got %s", contacts[0].Name)
	}

	if err := AddContact(contactsFile, "Bob", "age1pq1testkey456"); err != nil {
		t.Fatal(err)
	}
	contacts, _ = LoadContacts(contactsFile)
	if len(contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(contacts))
	}

	if err := RemoveContact(contactsFile, "Alice"); err != nil {
		t.Fatal(err)
	}
	contacts, _ = LoadContacts(contactsFile)
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact after removal, got %d", len(contacts))
	}
	if contacts[0].Name != "Bob" {
		t.Errorf("expected Bob remaining, got %s", contacts[0].Name)
	}
}

func TestShareAndImport(t *testing.T) {
	tmpDir := t.TempDir()

	senderKeyPath := filepath.Join(tmpDir, "sender-key.txt")
	senderIdentity, err := crypto.GenerateKey(senderKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	recipientKeyPath := filepath.Join(tmpDir, "recipient-key.txt")
	recipientIdentity, err := crypto.GenerateKey(recipientKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("---\ncustomer: Test\ntype: task\n---\n# Shared Note")
	notePath := filepath.Join(tmpDir, "original.md.age")
	if err := crypto.EncryptToFile(notePath, plaintext, senderIdentity.Recipient()); err != nil {
		t.Fatal(err)
	}

	exportDir := filepath.Join(tmpDir, "exports")
	os.MkdirAll(exportDir, 0700)

	exportPath, err := ShareNote(notePath, senderIdentity, recipientIdentity.Recipient(), exportDir)
	if err != nil {
		t.Fatalf("ShareNote: %v", err)
	}

	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("export file not created: %v", err)
	}

	imported, err := crypto.DecryptFile(exportPath, recipientIdentity)
	if err != nil {
		t.Fatalf("recipient cannot decrypt shared note: %v", err)
	}
	if string(imported) != string(plaintext) {
		t.Errorf("imported content mismatch")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/share/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write contacts.go**

```go
package share

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Contact struct {
	Name      string `yaml:"name"`
	PublicKey string `yaml:"public_key"`
}

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

func AddContact(path, name, publicKey string) error {
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

- [ ] **Step 4: Write share.go**

```go
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
```

- [ ] **Step 5: Run tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/share/ -v
```
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/share/
git commit -m "feat: add contact management and note sharing with re-encryption"
```

---

### Task 11: TUI Styles & App Shell

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/app.go`

- [ ] **Step 1: Create styles.go with Lip Gloss style definitions**

```go
package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6600")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5F5FD7"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	todayStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF8800")).
			Bold(true)

	upcomingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00"))

	urgentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	highStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF8800")).
			Bold(true)

	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87CEEB"))

	typeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98FB98"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5F5FD7"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	previewBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#5F5FD7")).
				Padding(0, 1)
)
```

- [ ] **Step 2: Create app.go — root Bubble Tea model**

```go
package tui

import (
	"filippo.io/age"
	tea "charm.land/bubbletea/v2"
	"github.com/kborup-redhat/pq-notes/internal/calendar"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type viewState int

const (
	viewDashboard viewState = iota
	viewSetup
	viewNewNote
	viewSearch
	viewFilter
)

type focusPane int

const (
	focusList focusPane = iota
	focusPreview
)

type App struct {
	cfg      *config.Config
	store    *notes.NoteStore
	cal      *calendar.BusinessCal
	identity *age.HybridIdentity

	notes    []*notes.Note
	cursor   int
	focus    focusPane
	view     viewState
	width    int
	height   int
	showDone bool
	err      error

	setup    *SetupModel
	newNote  *NewNoteModel
}

func NewApp(cfg *config.Config, store *notes.NoteStore, cal *calendar.BusinessCal, identity *age.HybridIdentity) *App {
	return &App{
		cfg:      cfg,
		store:    store,
		cal:      cal,
		identity: identity,
		view:     viewDashboard,
		focus:    focusList,
	}
}

func (a *App) Init() tea.Cmd {
	return a.loadNotes
}

func (a *App) loadNotes() tea.Msg {
	allNotes, err := a.store.List()
	if err != nil {
		return errMsg{err}
	}
	return notesLoadedMsg{notes: allNotes}
}

type notesLoadedMsg struct {
	notes []*notes.Note
}

type errMsg struct {
	err error
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case notesLoadedMsg:
		a.notes = msg.notes
		return a, nil

	case errMsg:
		a.err = msg.err
		return a, nil

	case tea.KeyPressMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()

	switch key.Code {
	case tea.KeyUp:
		if a.cursor > 0 {
			a.cursor--
		}
	case tea.KeyDown:
		if a.cursor < len(a.notes)-1 {
			a.cursor++
		}
	case tea.KeyTab:
		if a.focus == focusList {
			a.focus = focusPreview
		} else {
			a.focus = focusList
		}
	case 'q':
		return a, tea.Quit
	case 'a':
		a.showDone = !a.showDone
	}

	return a, nil
}

func (a *App) View() tea.View {
	if a.width == 0 {
		return tea.NewView("Loading...")
	}

	listWidth := a.width / 3
	previewWidth := a.width - listWidth - 3

	list := a.renderList(listWidth)
	preview := a.renderPreview(previewWidth)

	help := helpStyle.Render("[n]ew  [e]dit  [d]ue  [t]ag filter  [s]earch  [m]ark done  [q]uit")

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		borderStyle.Width(listWidth).Height(a.height-3).Render(list),
		previewBorderStyle.Width(previewWidth).Height(a.height-3).Render(preview),
	)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, content, help))
}

func (a *App) renderList(width int) string {
	if len(a.notes) == 0 {
		return headerStyle.Render("No notes yet. Press [n] to create one.")
	}

	var s string
	s += headerStyle.Render("NOTES") + "\n\n"

	for i, note := range a.notes {
		line := typeStyle.Render("["+string(note.Type)+"]") + " " + note.Title
		if i == a.cursor {
			s += selectedStyle.Width(width - 2).Render(line) + "\n"
		} else {
			s += normalStyle.Render(line) + "\n"
		}
	}
	return s
}

func (a *App) renderPreview(width int) string {
	if len(a.notes) == 0 || a.cursor >= len(a.notes) {
		return dimStyle.Render("Select a note to preview")
	}

	note := a.notes[a.cursor]
	return headerStyle.Render(note.Title) + "\n\n" +
		"Customer: " + note.Customer + "\n" +
		"Type: " + string(note.Type) + "\n" +
		"Status: " + string(note.Status) + "\n\n" +
		note.Body
}

func RunApp(cfg *config.Config, store *notes.NoteStore, cal *calendar.BusinessCal, identity *age.HybridIdentity) error {
	app := NewApp(cfg, store, cal, identity)
	p := tea.NewProgram(app)
	_, err := p.Run()
	return err
}
```

- [ ] **Step 3: Add Bubble Tea, Lip Gloss dependencies and verify build**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go get charm.land/bubbletea/v2@latest
go get charm.land/lipgloss/v2@latest
go get charm.land/glamour/v2@latest
go build ./...
```
Expected: builds successfully

- [ ] **Step 4: Commit**

```bash
git add internal/tui/ go.mod go.sum
git commit -m "feat: add TUI app shell with split-pane layout and Lip Gloss styles"
```

---

### Task 12: First-Launch Setup Wizard

**Files:**
- Create: `internal/tui/setup.go`
- Modify: `internal/tui/app.go` (integrate setup)
- Modify: `cmd/root.go` (check for config, launch setup or TUI)

- [ ] **Step 1: Create setup.go — 5-step TUI wizard**

```go
package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

type setupStep int

const (
	stepKey setupStep = iota
	stepEditor
	stepDateFormat
	stepCountry
	stepWeekend
	stepDone
)

type SetupModel struct {
	step        setupStep
	cfg         *config.Config
	notesDir    string
	configDir   string
	keyPath     string
	nameInput   string
	emailInput  string
	countryInput string
	editorChoice int
	dateChoice   int
	weekendDays  [7]bool
	inputFocused string
	err         error
	width       int
	height      int
}

func NewSetupModel(notesDir, configDir string) *SetupModel {
	return &SetupModel{
		step:      stepKey,
		cfg:       &config.Config{},
		notesDir:  notesDir,
		configDir: configDir,
		keyPath:   configDir + "/key.txt",
		weekendDays: [7]bool{false, false, false, false, false, true, true}, // sat, sun
		inputFocused: "name",
	}
}

func (m *SetupModel) Init() tea.Cmd {
	return nil
}

func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *SetupModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()

	switch key.Code {
	case tea.KeyEscape:
		return m, tea.Quit
	}

	switch m.step {
	case stepKey:
		return m.handleKeyStep(msg)
	case stepEditor:
		return m.handleEditorStep(msg)
	case stepDateFormat:
		return m.handleDateFormatStep(msg)
	case stepCountry:
		return m.handleCountryStep(msg)
	case stepWeekend:
		return m.handleWeekendStep(msg)
	}

	return m, nil
}

func (m *SetupModel) handleKeyStep(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()

	switch key.Code {
	case tea.KeyEnter:
		if m.inputFocused == "name" && m.nameInput != "" {
			m.inputFocused = "email"
		} else if m.inputFocused == "email" && m.emailInput != "" {
			_, err := crypto.GenerateKey(m.keyPath)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.step = stepEditor
		}
	case tea.KeyBackspace:
		if m.inputFocused == "name" && len(m.nameInput) > 0 {
			m.nameInput = m.nameInput[:len(m.nameInput)-1]
		} else if m.inputFocused == "email" && len(m.emailInput) > 0 {
			m.emailInput = m.emailInput[:len(m.emailInput)-1]
		}
	default:
		if key.Text != "" {
			if m.inputFocused == "name" {
				m.nameInput += key.Text
			} else {
				m.emailInput += key.Text
			}
		}
	}
	return m, nil
}

func (m *SetupModel) handleEditorStep(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()
	switch key.Code {
	case tea.KeyUp:
		if m.editorChoice > 0 {
			m.editorChoice--
		}
	case tea.KeyDown:
		if m.editorChoice < 1 {
			m.editorChoice++
		}
	case tea.KeyEnter:
		if m.editorChoice == 0 {
			m.cfg.Editor = "vi"
		} else {
			m.cfg.Editor = "code"
		}
		m.step = stepDateFormat
	}
	return m, nil
}

func (m *SetupModel) handleDateFormatStep(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()
	switch key.Code {
	case tea.KeyUp:
		if m.dateChoice > 0 {
			m.dateChoice--
		}
	case tea.KeyDown:
		if m.dateChoice < 1 {
			m.dateChoice++
		}
	case tea.KeyEnter:
		if m.dateChoice == 0 {
			m.cfg.DateFormat = "eu"
		} else {
			m.cfg.DateFormat = "us"
		}
		m.step = stepCountry
	}
	return m, nil
}

func (m *SetupModel) handleCountryStep(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()
	switch key.Code {
	case tea.KeyEnter:
		if m.countryInput != "" {
			m.cfg.Country = strings.ToUpper(m.countryInput)
			weekend := config.DefaultWeekend(m.cfg.Country)
			m.weekendDays = weekendToBools(weekend)
			m.step = stepWeekend
		}
	case tea.KeyBackspace:
		if len(m.countryInput) > 0 {
			m.countryInput = m.countryInput[:len(m.countryInput)-1]
		}
	default:
		if key.Text != "" && len(m.countryInput) < 2 {
			m.countryInput += key.Text
		}
	}
	return m, nil
}

func (m *SetupModel) handleWeekendStep(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()
	switch key.Code {
	case tea.KeyUp:
		// navigate days (cursor tracked via editorChoice reuse)
		if m.editorChoice > 0 {
			m.editorChoice--
		}
	case tea.KeyDown:
		if m.editorChoice < 6 {
			m.editorChoice++
		}
	case ' ':
		m.weekendDays[m.editorChoice] = !m.weekendDays[m.editorChoice]
	case tea.KeyEnter:
		m.cfg.Weekend = boolsToWeekend(m.weekendDays)
		m.step = stepDone
		return m, m.saveConfig
	}
	return m, nil
}

func (m *SetupModel) saveConfig() tea.Msg {
	if err := config.Save(m.cfg, m.configDir); err != nil {
		return errMsg{err}
	}
	return setupCompleteMsg{}
}

type setupCompleteMsg struct{}

func (m *SetupModel) View() tea.View {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Welcome to pq-notes") + "\n\n")
	s.WriteString("Let's get you set up.\n\n")

	switch m.step {
	case stepKey:
		s.WriteString(headerStyle.Render("Step 1/5 — Encryption Key") + "\n\n")
		if m.inputFocused == "name" {
			s.WriteString(fmt.Sprintf("  Name:  %s|\n", m.nameInput))
			s.WriteString(fmt.Sprintf("  Email: %s\n", m.emailInput))
		} else {
			s.WriteString(fmt.Sprintf("  Name:  %s\n", m.nameInput))
			s.WriteString(fmt.Sprintf("  Email: %s|\n", m.emailInput))
		}

	case stepEditor:
		s.WriteString(headerStyle.Render("Step 2/5 — Editor") + "\n\n")
		editors := []string{"vi", "code (VS Code)"}
		for i, e := range editors {
			cursor := "  "
			if i == m.editorChoice {
				cursor = "> "
			}
			s.WriteString(fmt.Sprintf("  %s%s\n", cursor, e))
		}

	case stepDateFormat:
		s.WriteString(headerStyle.Render("Step 3/5 — Date Format") + "\n\n")
		formats := []string{"EU (DD-MM-YYYY)", "US (MM-DD-YYYY)"}
		for i, f := range formats {
			cursor := "  "
			if i == m.dateChoice {
				cursor = "> "
			}
			s.WriteString(fmt.Sprintf("  %s%s\n", cursor, f))
		}

	case stepCountry:
		s.WriteString(headerStyle.Render("Step 4/5 — Country") + "\n\n")
		s.WriteString(fmt.Sprintf("  Country code (ISO): %s|\n", m.countryInput))
		s.WriteString("  (e.g. DK, SA, US, GB, DE)\n")

	case stepWeekend:
		s.WriteString(headerStyle.Render("Step 5/5 — Weekend Days") + "\n\n")
		days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
		for i, d := range days {
			cursor := "  "
			if i == m.editorChoice {
				cursor = "> "
			}
			check := "[ ]"
			if m.weekendDays[i] {
				check = "[x]"
			}
			s.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, check, d))
		}
		s.WriteString("\n  Space to toggle, Enter to confirm\n")

	case stepDone:
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("  Setup complete!") + "\n")
		s.WriteString("  Press Enter to start...\n")
	}

	if m.err != nil {
		s.WriteString("\n" + overdueStyle.Render("Error: "+m.err.Error()) + "\n")
	}

	return tea.NewView(s.String())
}

func weekendToBools(weekend []string) [7]bool {
	var bools [7]bool
	dayMap := map[string]int{
		"monday": 0, "tuesday": 1, "wednesday": 2, "thursday": 3,
		"friday": 4, "saturday": 5, "sunday": 6,
	}
	for _, d := range weekend {
		if idx, ok := dayMap[strings.ToLower(d)]; ok {
			bools[idx] = true
		}
	}
	return bools
}

func boolsToWeekend(bools [7]bool) []string {
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	var weekend []string
	for i, b := range bools {
		if b {
			weekend = append(weekend, days[i])
		}
	}
	return weekend
}
```

- [ ] **Step 2: Update cmd/root.go to integrate setup wizard and TUI launch**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/calendar"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
	"github.com/kborup-redhat/pq-notes/internal/notes"
	"github.com/kborup-redhat/pq-notes/internal/tui"
	tea "charm.land/bubbletea/v2"
)

var rootCmd = &cobra.Command{
	Use:   "pq-notes",
	Short: "Post-quantum encrypted terminal notes manager",
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir()
		configDir := config.ConfigDirIn(notesDir)

		if !config.Exists(configDir) {
			if err := os.MkdirAll(configDir, 0700); err != nil {
				return fmt.Errorf("create config dir: %w", err)
			}
			setup := tui.NewSetupModel(notesDir, configDir)
			p := tea.NewProgram(setup)
			if _, err := p.Run(); err != nil {
				return err
			}
		}

		cfg, err := config.Load(configDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		keyPath := configDir + "/key.txt"
		identity, err := crypto.LoadIdentity(keyPath)
		if err != nil {
			return fmt.Errorf("load key: %w", err)
		}

		store := notes.NewNoteStore(notesDir, identity, cfg.DateFormat)
		cal := calendar.New(cfg)

		return tui.RunApp(cfg, store, cal, identity)
	},
}

func Execute() error {
	return rootCmd.Execute()
}
```

- [ ] **Step 3: Build and manually test the setup wizard**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go build -o pq-notes .
./pq-notes
```
Expected: the setup wizard starts (Step 1/5), you can type name/email, navigate steps with arrow keys, and on completion it creates `~/notes/.pq-notes/config.yaml` and `key.txt`, then launches the empty dashboard.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/setup.go cmd/root.go
git commit -m "feat: add first-launch setup wizard with 5-step TUI flow"
```

---

### Task 13: Dashboard View with Backlog Sorting

**Files:**
- Create: `internal/tui/dashboard.go`
- Modify: `internal/tui/app.go` (use dashboard in view)

- [ ] **Step 1: Create dashboard.go — backlog sorted by due date with urgency sections**

```go
package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/kborup-redhat/pq-notes/internal/dateutil"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type urgencyGroup int

const (
	groupOverdue urgencyGroup = iota
	groupToday
	groupUpcoming
	groupNoDue
)

type DashboardItem struct {
	Note    *notes.Note
	Group   urgencyGroup
	Index   int
}

func BuildDashboard(allNotes []*notes.Note, showDone bool, dateFormat string) []DashboardItem {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)

	var items []DashboardItem
	for _, n := range allNotes {
		if !showDone && n.Status == notes.StatusDone {
			continue
		}

		group := groupNoDue
		if !n.Due.IsZero() {
			if n.Due.Before(today) {
				group = groupOverdue
			} else if n.Due.Before(tomorrow) {
				group = groupToday
			} else {
				group = groupUpcoming
			}
		}

		items = append(items, DashboardItem{Note: n, Group: group})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Group != items[j].Group {
			return items[i].Group < items[j].Group
		}
		if items[i].Note.Due.IsZero() && items[j].Note.Due.IsZero() {
			return items[i].Note.Created.After(items[j].Note.Created)
		}
		if items[i].Note.Due.IsZero() {
			return false
		}
		if items[j].Note.Due.IsZero() {
			return true
		}
		return items[i].Note.Due.Before(items[j].Note.Due)
	})

	for i := range items {
		items[i].Index = i
	}

	return items
}

func RenderDashboard(items []DashboardItem, cursor int, width int, dateFormat string) string {
	if len(items) == 0 {
		return headerStyle.Render("No notes yet. Press [n] to create one.")
	}

	var sb strings.Builder
	currentGroup := urgencyGroup(-1)

	groupHeaders := map[urgencyGroup]string{
		groupOverdue:  "OVERDUE",
		groupToday:    "TODAY",
		groupUpcoming: "UPCOMING",
		groupNoDue:    "NO DUE DATE",
	}

	groupStyles := map[urgencyGroup]lipgloss.Style{
		groupOverdue:  overdueStyle,
		groupToday:    todayStyle,
		groupUpcoming: upcomingStyle,
		groupNoDue:    dimStyle,
	}

	for _, item := range items {
		if item.Group != currentGroup {
			currentGroup = item.Group
			style := groupStyles[currentGroup]
			sb.WriteString("\n " + style.Render(groupHeaders[currentGroup]) + "\n")
		}

		line := renderDashboardItem(item, dateFormat)

		if item.Index == cursor {
			sb.WriteString(selectedStyle.Width(width - 2).Render(line) + "\n")
		} else if item.Note.Status == notes.StatusDone {
			sb.WriteString(dimStyle.Render(line) + "\n")
		} else {
			sb.WriteString(normalStyle.Render(line) + "\n")
		}
	}

	return sb.String()
}

func renderDashboardItem(item DashboardItem, dateFormat string) string {
	n := item.Note
	typeLabel := typeStyle.Render("[" + string(n.Type) + "]")

	var priority string
	switch n.Priority {
	case notes.PriorityUrgent:
		priority = urgentStyle.Render(" [URGENT]")
	case notes.PriorityHigh:
		priority = highStyle.Render(" [HIGH]")
	}

	dueStr := ""
	if !n.Due.IsZero() {
		dueStr = " Due: " + dateutil.FormatDateOnly(n.Due, dateFormat)
	}

	return fmt.Sprintf("  %s %s%s  %s%s", typeLabel, n.Customer, priority, n.Title, dueStr)
}
```

- [ ] **Step 2: Update app.go to use dashboard rendering**

Replace the `renderList` method in `internal/tui/app.go`:

```go
func (a *App) renderList(width int) string {
	items := BuildDashboard(a.notes, a.showDone, a.cfg.DateFormat)
	a.dashboardItems = items
	return RenderDashboard(items, a.cursor, width, a.cfg.DateFormat)
}
```

Add `dashboardItems []DashboardItem` field to the `App` struct. Update `handleKey` to use `len(a.dashboardItems)` instead of `len(a.notes)` for cursor bounds. Update `renderPreview` to use `a.dashboardItems[a.cursor].Note` instead of `a.notes[a.cursor]`.

- [ ] **Step 3: Build and test dashboard rendering**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go build -o pq-notes .
```
Expected: builds successfully

- [ ] **Step 4: Commit**

```bash
git add internal/tui/dashboard.go internal/tui/app.go
git commit -m "feat: add dashboard view with backlog sorted by urgency groups"
```

---

### Task 14: Markdown Preview Pane with Glamour

**Files:**
- Create: `internal/tui/preview.go`
- Modify: `internal/tui/app.go` (use preview rendering)

- [ ] **Step 1: Create preview.go — Glamour-based markdown rendering**

```go
package tui

import (
	"fmt"
	"strings"

	"charm.land/glamour/v2"
	"github.com/kborup-redhat/pq-notes/internal/dateutil"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

func RenderPreview(note *notes.Note, width int, dateFormat string) string {
	if note == nil {
		return dimStyle.Render("Select a note to preview")
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**Customer:** %s\n\n", note.Customer))
	sb.WriteString(fmt.Sprintf("**Type:** %s\n\n", note.Type))

	if !note.Created.IsZero() {
		sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", dateutil.FormatDate(note.Created, dateFormat)))
	}
	if !note.Due.IsZero() {
		sb.WriteString(fmt.Sprintf("**Due:** %s\n\n", dateutil.FormatDate(note.Due, dateFormat)))
	}
	if note.Repeat != "" {
		sb.WriteString(fmt.Sprintf("**Repeat:** %s\n\n", note.Repeat))
	}
	if len(note.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("**Tags:** %s\n\n", formatTags(note.Tags)))
	}
	if note.Priority != "" && note.Priority != notes.PriorityNormal {
		sb.WriteString(fmt.Sprintf("**Priority:** %s\n\n", note.Priority))
	}
	if len(note.Attendees) > 0 {
		sb.WriteString(fmt.Sprintf("**Attendees:** %s\n\n", strings.Join(note.Attendees, ", ")))
	}

	sb.WriteString("---\n\n")
	sb.WriteString(note.Body)

	r, err := glamour.NewTermRenderer(glamour.WithWordWrap(width - 4))
	if err != nil {
		return sb.String()
	}

	rendered, err := r.Render(sb.String())
	if err != nil {
		return sb.String()
	}

	return rendered
}

func formatTags(tags []string) string {
	var formatted []string
	for _, t := range tags {
		formatted = append(formatted, "#"+t)
	}
	return strings.Join(formatted, " ")
}
```

- [ ] **Step 2: Update app.go renderPreview to use Glamour**

Replace the `renderPreview` method in `internal/tui/app.go`:

```go
func (a *App) renderPreview(width int) string {
	if len(a.dashboardItems) == 0 || a.cursor >= len(a.dashboardItems) {
		return dimStyle.Render("Select a note to preview")
	}
	note := a.dashboardItems[a.cursor].Note
	return RenderPreview(note, width, a.cfg.DateFormat)
}
```

- [ ] **Step 3: Build and verify**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go build -o pq-notes .
```
Expected: builds successfully

- [ ] **Step 4: Commit**

```bash
git add internal/tui/preview.go internal/tui/app.go
git commit -m "feat: add Glamour markdown preview pane"
```

---

### Task 15: New Note Wizard (TUI-driven metadata collection)

**Files:**
- Create: `internal/tui/newnote.go`
- Modify: `internal/tui/app.go` (wire up 'n' key)

- [ ] **Step 1: Create newnote.go — multi-step note creation wizard**

```go
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/dateutil"
	"github.com/kborup-redhat/pq-notes/internal/editor"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type newNoteStep int

const (
	nnStepType newNoteStep = iota
	nnStepCustomer
	nnStepTitle
	nnStepDue
	nnStepRepeat
	nnStepTags
	nnStepPriority
	nnStepAttendees
	nnStepRelated
	nnStepConfirm
)

type NewNoteModel struct {
	step          newNoteStep
	cfg           *config.Config
	store         *notes.NoteStore
	existingNotes []*notes.Note

	typeChoice    int
	customerInput string
	titleInput    string
	dueInput      string
	repeatChoice  int
	customRepeat  string
	tagsInput     string
	priorityChoice int
	attendeesInput string
	relatedChoice  int

	err error
}

func NewNewNoteModel(cfg *config.Config, store *notes.NoteStore, existing []*notes.Note) *NewNoteModel {
	return &NewNoteModel{
		step:          nnStepType,
		cfg:           cfg,
		store:         store,
		existingNotes: existing,
	}
}

func (m *NewNoteModel) noteType() notes.NoteType {
	types := []notes.NoteType{notes.Meeting, notes.Task, notes.Reminder, notes.Followup}
	return types[m.typeChoice]
}

func (m *NewNoteModel) Update(msg tea.KeyPressMsg) (done bool, cmd tea.Cmd) {
	key := msg.Key()

	if key.Code == tea.KeyEscape {
		return true, nil
	}

	switch m.step {
	case nnStepType:
		switch key.Code {
		case tea.KeyUp:
			if m.typeChoice > 0 { m.typeChoice-- }
		case tea.KeyDown:
			if m.typeChoice < 3 { m.typeChoice++ }
		case tea.KeyEnter:
			m.step = nnStepCustomer
		}

	case nnStepCustomer:
		switch key.Code {
		case tea.KeyEnter:
			if m.customerInput != "" { m.step = nnStepTitle }
		case tea.KeyBackspace:
			if len(m.customerInput) > 0 { m.customerInput = m.customerInput[:len(m.customerInput)-1] }
		default:
			if key.Text != "" { m.customerInput += key.Text }
		}

	case nnStepTitle:
		switch key.Code {
		case tea.KeyEnter:
			if m.titleInput != "" { m.step = nnStepDue }
		case tea.KeyBackspace:
			if len(m.titleInput) > 0 { m.titleInput = m.titleInput[:len(m.titleInput)-1] }
		default:
			if key.Text != "" { m.titleInput += key.Text }
		}

	case nnStepDue:
		switch key.Code {
		case tea.KeyEnter:
			nextStep := nnStepTags
			if m.dueInput != "" && m.dueInput != "none" {
				nextStep = nnStepRepeat
			}
			if m.noteType() == notes.Task {
				nextStep = nnStepPriority
				if m.dueInput != "" && m.dueInput != "none" {
					nextStep = nnStepRepeat
				}
			}
			m.step = nextStep
		case tea.KeyBackspace:
			if len(m.dueInput) > 0 { m.dueInput = m.dueInput[:len(m.dueInput)-1] }
		default:
			if key.Text != "" { m.dueInput += key.Text }
		}

	case nnStepRepeat:
		switch key.Code {
		case tea.KeyUp:
			if m.repeatChoice > 0 { m.repeatChoice-- }
		case tea.KeyDown:
			if m.repeatChoice < 4 { m.repeatChoice++ }
		case tea.KeyEnter:
			m.step = m.nextAfterRepeat()
		}

	case nnStepTags:
		switch key.Code {
		case tea.KeyEnter:
			m.step = nnStepConfirm
			if m.noteType() == notes.Meeting { m.step = nnStepAttendees }
			if m.noteType() == notes.Followup { m.step = nnStepRelated }
		case tea.KeyBackspace:
			if len(m.tagsInput) > 0 { m.tagsInput = m.tagsInput[:len(m.tagsInput)-1] }
		default:
			if key.Text != "" { m.tagsInput += key.Text }
		}

	case nnStepPriority:
		switch key.Code {
		case tea.KeyUp:
			if m.priorityChoice > 0 { m.priorityChoice-- }
		case tea.KeyDown:
			if m.priorityChoice < 3 { m.priorityChoice++ }
		case tea.KeyEnter:
			m.step = nnStepTags
		}

	case nnStepAttendees:
		switch key.Code {
		case tea.KeyEnter:
			m.step = nnStepConfirm
		case tea.KeyBackspace:
			if len(m.attendeesInput) > 0 { m.attendeesInput = m.attendeesInput[:len(m.attendeesInput)-1] }
		default:
			if key.Text != "" { m.attendeesInput += key.Text }
		}

	case nnStepRelated:
		switch key.Code {
		case tea.KeyUp:
			if m.relatedChoice > 0 { m.relatedChoice-- }
		case tea.KeyDown:
			if m.relatedChoice < len(m.existingNotes) { m.relatedChoice++ }
		case tea.KeyEnter:
			m.step = nnStepConfirm
		}

	case nnStepConfirm:
		switch key.Code {
		case tea.KeyEnter:
			return true, m.createNote
		}
	}

	return false, nil
}

func (m *NewNoteModel) nextAfterRepeat() newNoteStep {
	if m.noteType() == notes.Task {
		return nnStepPriority
	}
	return nnStepTags
}

func (m *NewNoteModel) createNote() tea.Msg {
	now := time.Now()
	note := &notes.Note{
		Customer: m.customerInput,
		Type:     m.noteType(),
		Created:  now,
		Title:    m.titleInput,
		Status:   notes.StatusOpen,
	}

	if m.dueInput != "" && m.dueInput != "none" {
		due, err := dateutil.ParseDate(m.dueInput, m.cfg.DateFormat, now)
		if err == nil {
			note.Due = due
		}
	}

	repeatOptions := []string{"", "daily", "weekly", "monthly", "custom"}
	if m.repeatChoice > 0 && m.repeatChoice < len(repeatOptions) {
		note.Repeat = repeatOptions[m.repeatChoice]
		if note.Repeat == "custom" {
			note.Repeat = m.customRepeat
		}
	}

	if m.tagsInput != "" {
		for _, t := range strings.Split(m.tagsInput, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				note.Tags = append(note.Tags, t)
			}
		}
	}

	priorities := []notes.Priority{notes.PriorityLow, notes.PriorityNormal, notes.PriorityHigh, notes.PriorityUrgent}
	if m.noteType() == notes.Task {
		note.Priority = priorities[m.priorityChoice]
	}

	if m.attendeesInput != "" {
		for _, a := range strings.Split(m.attendeesInput, ",") {
			a = strings.TrimSpace(a)
			if a != "" {
				note.Attendees = append(note.Attendees, a)
			}
		}
	}

	if m.noteType() == notes.Followup && m.relatedChoice > 0 && m.relatedChoice <= len(m.existingNotes) {
		related := m.existingNotes[m.relatedChoice-1]
		note.Related = notes.NoteFilename(related.Title, related.Created)
	}

	path, err := m.store.Create(note)
	if err != nil {
		return errMsg{err}
	}

	return noteCreatedMsg{path: path}
}

type noteCreatedMsg struct {
	path string
}

func (m *NewNoteModel) View() string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("New Note") + "\n\n")

	switch m.step {
	case nnStepType:
		sb.WriteString("  Type:\n")
		types := []string{"Meeting", "Task", "Reminder", "Follow-up"}
		for i, t := range types {
			cursor := "  "
			if i == m.typeChoice { cursor = "> " }
			sb.WriteString(fmt.Sprintf("  %s%s\n", cursor, t))
		}

	case nnStepCustomer:
		sb.WriteString(fmt.Sprintf("  Customer: %s|\n", m.customerInput))

	case nnStepTitle:
		sb.WriteString(fmt.Sprintf("  Title: %s|\n", m.titleInput))

	case nnStepDue:
		sb.WriteString(fmt.Sprintf("  Due: %s|\n", m.dueInput))
		sb.WriteString("  (DD-MM-YYYY, tomorrow, friday, none)\n")

	case nnStepRepeat:
		sb.WriteString("  Repeat:\n")
		options := []string{"None", "Daily", "Weekly", "Monthly", "Custom..."}
		for i, o := range options {
			cursor := "  "
			if i == m.repeatChoice { cursor = "> " }
			sb.WriteString(fmt.Sprintf("  %s%s\n", cursor, o))
		}

	case nnStepTags:
		sb.WriteString(fmt.Sprintf("  Tags (comma-separated): %s|\n", m.tagsInput))

	case nnStepPriority:
		sb.WriteString("  Priority:\n")
		priorities := []string{"Low", "Normal", "High", "Urgent"}
		for i, p := range priorities {
			cursor := "  "
			if i == m.priorityChoice { cursor = "> " }
			sb.WriteString(fmt.Sprintf("  %s%s\n", cursor, p))
		}

	case nnStepAttendees:
		sb.WriteString(fmt.Sprintf("  Attendees (comma-separated): %s|\n", m.attendeesInput))

	case nnStepRelated:
		sb.WriteString("  Related note:\n")
		sb.WriteString("  > (none)\n")
		for i, n := range m.existingNotes {
			cursor := "  "
			if i+1 == m.relatedChoice { cursor = "> " }
			sb.WriteString(fmt.Sprintf("  %s%s - %s\n", cursor, n.Customer, n.Title))
		}

	case nnStepConfirm:
		sb.WriteString("  Summary:\n")
		sb.WriteString(fmt.Sprintf("    Type:     %s\n", m.noteType()))
		sb.WriteString(fmt.Sprintf("    Customer: %s\n", m.customerInput))
		sb.WriteString(fmt.Sprintf("    Title:    %s\n", m.titleInput))
		sb.WriteString(fmt.Sprintf("    Due:      %s\n", m.dueInput))
		sb.WriteString(fmt.Sprintf("    Tags:     %s\n", m.tagsInput))
		sb.WriteString("\n  Press Enter to create, Esc to cancel\n")
	}

	return sb.String()
}
```

- [ ] **Step 2: Wire up 'n' key in app.go**

Add to the `handleKey` method in `app.go`, inside the switch on `key.Code`:

```go
	case 'n':
		a.newNote = NewNewNoteModel(a.cfg, a.store, a.notes)
		a.view = viewNewNote
```

Add handling for `viewNewNote` in the `Update` method:

```go
	if a.view == viewNewNote && a.newNote != nil {
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			done, cmd := a.newNote.Update(kp)
			if done {
				a.view = viewDashboard
				a.newNote = nil
				return a, tea.Batch(cmd, a.loadNotes)
			}
			return a, cmd
		}
	}
```

Add handling for `noteCreatedMsg`:

```go
	case noteCreatedMsg:
		// Open in editor
		return a, func() tea.Msg {
			if err := editor.Open(a.cfg.Editor, msg.path); err != nil {
				return errMsg{err}
			}
			return editorClosedMsg{path: msg.path}
		}

	case editorClosedMsg:
		return a, a.loadNotes
```

Add `editorClosedMsg`:

```go
type editorClosedMsg struct {
	path string
}
```

Update `View()` to render the new note wizard when active:

```go
	if a.view == viewNewNote && a.newNote != nil {
		return tea.NewView(a.newNote.View())
	}
```

- [ ] **Step 3: Build and test new note creation**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go build -o pq-notes .
./pq-notes
```
Expected: press `n` to open new note wizard, fill in fields, confirm, editor opens with pre-filled template, on close returns to dashboard with new note visible.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/newnote.go internal/tui/app.go
git commit -m "feat: add TUI-driven new note creation wizard with all metadata fields"
```

---

### Task 16: Search & Filter Overlays

**Files:**
- Create: `internal/tui/search.go`
- Create: `internal/tui/filter.go`
- Modify: `internal/tui/app.go` (wire up 's', 't', 'y' keys)

- [ ] **Step 1: Create search.go — fuzzy search overlay**

```go
package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type SearchModel struct {
	query   string
	results []*notes.Note
	cursor  int
	store   *notes.NoteStore
}

func NewSearchModel(store *notes.NoteStore) *SearchModel {
	return &SearchModel{store: store}
}

func (m *SearchModel) Update(msg tea.KeyPressMsg) (done bool, selected *notes.Note) {
	key := msg.Key()

	switch key.Code {
	case tea.KeyEscape:
		return true, nil
	case tea.KeyEnter:
		if m.cursor < len(m.results) {
			return true, m.results[m.cursor]
		}
		return true, nil
	case tea.KeyUp:
		if m.cursor > 0 { m.cursor-- }
	case tea.KeyDown:
		if m.cursor < len(m.results)-1 { m.cursor++ }
	case tea.KeyBackspace:
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.search()
		}
	default:
		if key.Text != "" {
			m.query += key.Text
			m.search()
		}
	}
	return false, nil
}

func (m *SearchModel) search() {
	if m.query == "" {
		m.results = nil
		return
	}
	m.results, _ = m.store.Search(m.query)
	m.cursor = 0
}

func (m *SearchModel) View() string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Search") + "\n")
	sb.WriteString("  > " + m.query + "|\n\n")

	if len(m.results) == 0 && m.query != "" {
		sb.WriteString(dimStyle.Render("  No results") + "\n")
	}

	for i, n := range m.results {
		line := typeStyle.Render("["+string(n.Type)+"]") + " " + n.Customer + " — " + n.Title
		if i == m.cursor {
			sb.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			sb.WriteString(normalStyle.Render(line) + "\n")
		}
	}

	sb.WriteString("\n" + dimStyle.Render("  Enter to select, Esc to cancel") + "\n")
	return sb.String()
}
```

- [ ] **Step 2: Create filter.go — tag and type filter overlays**

```go
package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

type FilterModel struct {
	items    []string
	selected map[string]bool
	cursor   int
	title    string
}

func NewTagFilter(allNotes []*notes.Note) *FilterModel {
	tagSet := make(map[string]bool)
	for _, n := range allNotes {
		for _, t := range n.Tags {
			tagSet[t] = true
		}
	}
	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	return &FilterModel{
		items:    tags,
		selected: make(map[string]bool),
		title:    "Filter by Tag",
	}
}

func NewTypeFilter() *FilterModel {
	return &FilterModel{
		items:    []string{"meeting", "task", "reminder", "followup"},
		selected: make(map[string]bool),
		title:    "Filter by Type",
	}
}

func (m *FilterModel) Update(msg tea.KeyPressMsg) (done bool) {
	key := msg.Key()
	switch key.Code {
	case tea.KeyEscape:
		return true
	case tea.KeyEnter:
		return true
	case tea.KeyUp:
		if m.cursor > 0 { m.cursor-- }
	case tea.KeyDown:
		if m.cursor < len(m.items)-1 { m.cursor++ }
	case ' ':
		item := m.items[m.cursor]
		m.selected[item] = !m.selected[item]
	}
	return false
}

func (m *FilterModel) SelectedItems() []string {
	var result []string
	for item, sel := range m.selected {
		if sel {
			result = append(result, item)
		}
	}
	return result
}

func (m *FilterModel) View() string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render(m.title) + "\n\n")

	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor { cursor = "> " }
		check := "[ ]"
		if m.selected[item] { check = "[x]" }
		sb.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, check, item))
	}

	sb.WriteString("\n" + dimStyle.Render("  Space to toggle, Enter to apply, Esc to cancel") + "\n")
	return sb.String()
}
```

- [ ] **Step 3: Wire up 's', 't', 'y' keys in app.go**

Add to the `handleKey` switch in `app.go`:

```go
	case 's':
		a.search = NewSearchModel(a.store)
		a.view = viewSearch
	case 't':
		a.tagFilter = NewTagFilter(a.notes)
		a.view = viewFilter
	case 'y':
		a.typeFilter = NewTypeFilter()
		a.view = viewFilter
```

Add fields to `App` struct:

```go
	search     *SearchModel
	tagFilter  *FilterModel
	typeFilter *FilterModel
	activeTagFilter  []string
	activeTypeFilter []string
```

Handle these views in `Update` and `View` methods accordingly.

- [ ] **Step 4: Build and test**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go build -o pq-notes .
```
Expected: builds successfully

- [ ] **Step 5: Commit**

```bash
git add internal/tui/search.go internal/tui/filter.go internal/tui/app.go
git commit -m "feat: add search and filter overlays for tags and note types"
```

---

### Task 17: CLI Subcommands (key, contacts, config, share)

**Files:**
- Create: `cmd/key.go`
- Create: `cmd/contacts.go`
- Create: `cmd/config_cmd.go`
- Create: `cmd/share.go`

- [ ] **Step 1: Create cmd/key.go**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage encryption keys",
}

var keyShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display your public recipient key",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		identity, err := crypto.LoadIdentity(configDir + "/key.txt")
		if err != nil {
			return fmt.Errorf("load key: %w", err)
		}
		fmt.Println(crypto.PublicKey(identity))
		return nil
	},
}

var keyExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export key to file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		private, _ := cmd.Flags().GetBool("private")

		if private {
			fmt.Fprintln(os.Stderr, "WARNING: This exports your PRIVATE key. Keep it safe!")
			data, err := os.ReadFile(configDir + "/key.txt")
			if err != nil {
				return err
			}
			outPath := "pq-notes-identity.txt"
			if err := os.WriteFile(outPath, data, 0600); err != nil {
				return err
			}
			fmt.Printf("Private key exported to %s\n", outPath)
		} else {
			identity, err := crypto.LoadIdentity(configDir + "/key.txt")
			if err != nil {
				return err
			}
			outPath := "pq-notes-public-key.txt"
			if err := os.WriteFile(outPath, []byte(crypto.PublicKey(identity)+"\n"), 0644); err != nil {
				return err
			}
			fmt.Printf("Public key exported to %s\n", outPath)
		}
		return nil
	},
}

var keyImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import an existing identity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return err
		}
		if err := os.WriteFile(configDir+"/key.txt", data, 0600); err != nil {
			return err
		}
		fmt.Println("Key imported successfully")
		return nil
	},
}

func init() {
	keyExportCmd.Flags().Bool("private", false, "Export private identity (use with caution)")
	keyCmd.AddCommand(keyShowCmd, keyExportCmd, keyImportCmd)
	rootCmd.AddCommand(keyCmd)
}
```

- [ ] **Step 2: Create cmd/contacts.go**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/share"
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage contacts for note sharing",
}

var contactsAddCmd = &cobra.Command{
	Use:   "add [name] [public-key]",
	Short: "Add a contact",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactsFile := config.ConfigDirIn(config.NotesDir()) + "/contacts.yaml"
		if err := share.AddContact(contactsFile, args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Added contact %q\n", args[0])
		return nil
	},
}

var contactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		contactsFile := config.ConfigDirIn(config.NotesDir()) + "/contacts.yaml"
		contacts, err := share.LoadContacts(contactsFile)
		if err != nil {
			return err
		}
		if len(contacts) == 0 {
			fmt.Println("No contacts saved")
			return nil
		}
		for _, c := range contacts {
			fmt.Printf("  %s: %s\n", c.Name, c.PublicKey)
		}
		return nil
	},
}

var contactsRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactsFile := config.ConfigDirIn(config.NotesDir()) + "/contacts.yaml"
		if err := share.RemoveContact(contactsFile, args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed contact %q\n", args[0])
		return nil
	},
}

func init() {
	contactsCmd.AddCommand(contactsAddCmd, contactsListCmd, contactsRemoveCmd)
	rootCmd.AddCommand(contactsCmd)
}
```

- [ ] **Step 3: Create cmd/share.go**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
	internalShare "github.com/kborup-redhat/pq-notes/internal/share"
)

var shareCmd = &cobra.Command{
	Use:   "share [note-path]",
	Short: "Export an encrypted note for sharing",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactName, _ := cmd.Flags().GetString("for")
		if contactName == "" {
			return fmt.Errorf("--for flag is required (contact name)")
		}

		configDir := config.ConfigDirIn(config.NotesDir())
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return fmt.Errorf("load identity: %w", err)
		}

		contact, err := internalShare.FindContact(filepath.Join(configDir, "contacts.yaml"), contactName)
		if err != nil {
			return err
		}

		recipient, err := age.ParseHybridRecipient(contact.PublicKey)
		if err != nil {
			return fmt.Errorf("parse recipient key: %w", err)
		}

		exportDir := filepath.Join(configDir, "exports")
		os.MkdirAll(exportDir, 0700)

		exportPath, err := internalShare.ShareNote(args[0], identity, recipient, exportDir)
		if err != nil {
			return err
		}

		fmt.Printf("Note exported to: %s\n", exportPath)
		fmt.Println("Send this file to your contact via email, Slack, etc.")
		return nil
	},
}

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import a shared note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return err
		}

		plaintext, err := crypto.DecryptFile(args[0], identity)
		if err != nil {
			return fmt.Errorf("cannot decrypt (wrong key?): %w", err)
		}

		filename := filepath.Base(args[0])
		destDir := filepath.Join(config.NotesDir(), "Shared")
		os.MkdirAll(destDir, 0700)
		destPath := filepath.Join(destDir, filename)

		if err := crypto.EncryptToFile(destPath, plaintext, identity.Recipient()); err != nil {
			return err
		}

		fmt.Printf("Note imported to: %s\n", destPath)
		return nil
	},
}

func init() {
	shareCmd.Flags().String("for", "", "Contact name to share with")
	rootCmd.AddCommand(shareCmd, importCmd)
}
```

- [ ] **Step 4: Create cmd/config_cmd.go**

```go
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	tea "charm.land/bubbletea/v2"
	"github.com/kborup-redhat/pq-notes/internal/tui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Reopen the setup wizard to change settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir()
		configDir := config.ConfigDirIn(notesDir)
		setup := tui.NewSetupModel(notesDir, configDir)
		p := tea.NewProgram(setup)
		_, err := p.Run()
		return err
	},
}

var holidaysCmd = &cobra.Command{
	Use:   "holidays",
	Short: "Manage custom holidays",
}

var holidaysAddCmd = &cobra.Command{
	Use:   "add [name] [DD-MM]",
	Short: "Add a custom recurring holiday",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		cfg.CustomHolidays = append(cfg.CustomHolidays, config.CustomHoliday{
			Name: args[0],
			Date: args[1],
		})
		if err := config.Save(cfg, configDir); err != nil {
			return err
		}
		fmt.Printf("Added holiday %q on %s\n", args[0], args[1])
		return nil
	},
}

var holidaysListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show configured holidays",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		if len(cfg.CustomHolidays) == 0 {
			fmt.Println("No custom holidays configured")
			return nil
		}
		for _, h := range cfg.CustomHolidays {
			fmt.Printf("  %s: %s\n", h.Name, h.Date)
		}
		return nil
	},
}

var holidaysRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a custom holiday",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		var filtered []config.CustomHoliday
		for _, h := range cfg.CustomHolidays {
			if h.Name != args[0] {
				filtered = append(filtered, h)
			}
		}
		cfg.CustomHolidays = filtered
		if err := config.Save(cfg, configDir); err != nil {
			return err
		}
		fmt.Printf("Removed holiday %q\n", args[0])
		return nil
	},
}

func init() {
	_ = filepath.Join // satisfy import
	holidaysCmd.AddCommand(holidaysAddCmd, holidaysListCmd, holidaysRemoveCmd)
	configCmd.AddCommand(holidaysCmd)
	rootCmd.AddCommand(configCmd)
}
```

- [ ] **Step 5: Build and test all CLI subcommands**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go build -o pq-notes .
./pq-notes key --help
./pq-notes contacts --help
./pq-notes share --help
./pq-notes config --help
./pq-notes config holidays --help
```
Expected: all subcommands show help text

- [ ] **Step 6: Commit**

```bash
git add cmd/
git commit -m "feat: add CLI subcommands for key, contacts, share, config, and holidays"
```

---

### Task 18: Notification Daemon

**Files:**
- Create: `internal/daemon/daemon.go`
- Create: `internal/daemon/notify.go`
- Create: `internal/daemon/notify_linux.go`
- Create: `internal/daemon/notify_windows.go`
- Create: `internal/daemon/notify_darwin.go`
- Create: `internal/daemon/install.go`
- Create: `internal/daemon/daemon_test.go`
- Create: `cmd/daemon.go`

- [ ] **Step 1: Write failing test for notification tracking**

```go
package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNotificationTracking(t *testing.T) {
	tmpDir := t.TempDir()
	tracker := NewTracker(filepath.Join(tmpDir, "notified.json"))

	noteID := "Acme-Corp/2026-05-06-meeting.md.age"

	if tracker.WasNotified(noteID) {
		t.Error("should not be notified yet")
	}

	tracker.MarkNotified(noteID)
	if err := tracker.Save(); err != nil {
		t.Fatal(err)
	}

	loaded := NewTracker(filepath.Join(tmpDir, "notified.json"))
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if !loaded.WasNotified(noteID) {
		t.Error("should be notified after load")
	}
}

func TestShouldNotify(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.Local)

	overdue := time.Date(2026, 5, 4, 17, 0, 0, 0, time.Local)
	if !ShouldNotify(overdue, now) {
		t.Error("overdue note should trigger notification")
	}

	dueNow := time.Date(2026, 5, 6, 10, 0, 0, 0, time.Local)
	if !ShouldNotify(dueNow, now) {
		t.Error("note due now should trigger notification")
	}

	future := time.Date(2026, 5, 10, 17, 0, 0, 0, time.Local)
	if ShouldNotify(future, now) {
		t.Error("future note should not trigger notification")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/daemon/ -v
```
Expected: FAIL — package does not exist

- [ ] **Step 3: Write daemon.go**

```go
package daemon

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"filippo.io/age"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/notes"
)

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

func (t *Tracker) WasNotified(noteID string) bool {
	_, ok := t.Notified[noteID]
	return ok
}

func (t *Tracker) MarkNotified(noteID string) {
	t.Notified[noteID] = time.Now()
}

func ShouldNotify(due, now time.Time) bool {
	return !due.IsZero() && !due.After(now)
}

func Run(cfg *config.Config, identity *age.HybridIdentity, notesDir, configDir string) {
	store := notes.NewNoteStore(notesDir, identity, cfg.DateFormat)
	tracker := NewTracker(configDir + "/notified.json")
	tracker.Load()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	check(store, tracker)

	for range ticker.C {
		check(store, tracker)
	}
}

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

		if err := SendNotification(n.Title, n.Customer, n.Due); err != nil {
			log.Printf("daemon: notify: %v", err)
			continue
		}
		tracker.MarkNotified(noteID)
	}

	tracker.Save()
}
```

- [ ] **Step 4: Write notify.go interface and platform-specific implementations**

`internal/daemon/notify.go`:
```go
package daemon

import "time"

func SendNotification(title, customer string, due time.Time) error {
	body := "Customer: " + customer + " — Due: " + due.Format("02-01-2006 15:04")
	return sendOSNotification("pq-notes: "+title, body)
}
```

`internal/daemon/notify_linux.go`:
```go
//go:build linux

package daemon

import "os/exec"

func sendOSNotification(title, body string) error {
	return exec.Command("notify-send", title, body).Run()
}
```

`internal/daemon/notify_darwin.go`:
```go
//go:build darwin

package daemon

import "os/exec"

func sendOSNotification(title, body string) error {
	script := `display notification "` + body + `" with title "` + title + `"`
	return exec.Command("osascript", "-e", script).Run()
}
```

`internal/daemon/notify_windows.go`:
```go
//go:build windows

package daemon

import "os/exec"

func sendOSNotification(title, body string) error {
	script := `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; ` +
		`$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); ` +
		`$textNodes = $template.GetElementsByTagName('text'); ` +
		`$textNodes.Item(0).AppendChild($template.CreateTextNode('` + title + `')) > $null; ` +
		`$textNodes.Item(1).AppendChild($template.CreateTextNode('` + body + `')) > $null; ` +
		`$toast = [Windows.UI.Notifications.ToastNotification]::new($template); ` +
		`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('pq-notes').Show($toast)`
	return exec.Command("powershell", "-Command", script).Run()
}
```

- [ ] **Step 5: Write install.go for daemon auto-start**

```go
package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func Install(binaryPath string) error {
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

func Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallSystemd()
	case "darwin":
		return uninstallLaunchd()
	case "windows":
		return uninstallScheduledTask()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
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

func installSystemd(binaryPath string) error {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	os.MkdirAll(dir, 0755)

	unit := fmt.Sprintf(`[Unit]
Description=pq-notes notification daemon

[Service]
ExecStart=%s daemon
Restart=always

[Install]
WantedBy=default.target
`, binaryPath)

	if err := os.WriteFile(filepath.Join(dir, "pq-notes.service"), []byte(unit), 0644); err != nil {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return exec.Command("systemctl", "--user", "enable", "--now", "pq-notes").Run()
}

func uninstallSystemd() error {
	exec.Command("systemctl", "--user", "disable", "--now", "pq-notes").Run()
	path := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "pq-notes.service")
	os.Remove(path)
	return exec.Command("systemctl", "--user", "daemon-reload").Run()
}

func installLaunchd(binaryPath string) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>com.pq-notes.daemon</string>
    <key>ProgramArguments</key><array><string>%s</string><string>daemon</string></array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
</dict>
</plist>`, binaryPath)

	path := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.pq-notes.daemon.plist")
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", path).Run()
}

func uninstallLaunchd() error {
	path := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.pq-notes.daemon.plist")
	exec.Command("launchctl", "unload", path).Run()
	return os.Remove(path)
}

func installScheduledTask(binaryPath string) error {
	return exec.Command("schtasks", "/create", "/sc", "onlogon", "/tn", "pq-notes-daemon",
		"/tr", binaryPath+" daemon", "/f").Run()
}

func uninstallScheduledTask() error {
	return exec.Command("schtasks", "/delete", "/tn", "pq-notes-daemon", "/f").Run()
}
```

- [ ] **Step 6: Create cmd/daemon.go**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
	"github.com/kborup-redhat/pq-notes/internal/daemon"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the notification daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir()
		configDir := config.ConfigDirIn(notesDir)

		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return err
		}

		fmt.Println("pq-notes daemon started")
		daemon.Run(cfg, identity, notesDir, configDir)
		return nil
	},
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install daemon as auto-start service",
	RunE: func(cmd *cobra.Command, args []string) error {
		binary, err := os.Executable()
		if err != nil {
			return err
		}
		if err := daemon.Install(binary); err != nil {
			return err
		}
		fmt.Println("Daemon installed and started")
		return nil
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove daemon auto-start service",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.Uninstall(); err != nil {
			return err
		}
		fmt.Println("Daemon uninstalled")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	RunE: func(cmd *cobra.Command, args []string) error {
		status, err := daemon.Status()
		if err != nil {
			fmt.Printf("Daemon status: not running (%v)\n", err)
			return nil
		}
		fmt.Printf("Daemon status: %s", status)
		return nil
	},
}

func init() {
	daemonCmd.AddCommand(daemonInstallCmd, daemonUninstallCmd, daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)
}
```

- [ ] **Step 7: Run tests and build**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./internal/daemon/ -v
go build -o pq-notes .
./pq-notes daemon --help
```
Expected: daemon tests pass, help text shows all subcommands

- [ ] **Step 8: Commit**

```bash
git add internal/daemon/ cmd/daemon.go
git commit -m "feat: add notification daemon with cross-platform support and auto-start install"
```

---

### Task 19: Google Drive Sync (stub)

**Files:**
- Create: `internal/drive/drive.go`
- Create: `cmd/drive.go`

This task creates the Drive sync structure and CLI commands. The actual Google Drive API OAuth2 flow and file sync logic requires a Google Cloud project with Drive API enabled and OAuth credentials, which must be set up by the user. The implementation provides the full framework but the OAuth client ID/secret must be configured.

- [ ] **Step 1: Create drive.go with setup, sync, and auto framework**

```go
package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

type DriveSync struct {
	service    *drive.Service
	identity   *age.HybridIdentity
	notesDir   string
	configDir  string
	folderID   string
}

func Setup(configDir string, identity *age.HybridIdentity) error {
	fmt.Println("Google Drive Setup")
	fmt.Println("==================")
	fmt.Println("To use Drive sync, you need a Google Cloud project with the Drive API enabled.")
	fmt.Println("1. Go to https://console.cloud.google.com/apis/credentials")
	fmt.Println("2. Create an OAuth 2.0 Client ID (Desktop application)")
	fmt.Println("3. Download the JSON credentials file")
	fmt.Print("\nPath to credentials JSON: ")

	var credPath string
	fmt.Scanln(&credPath)

	credData, err := os.ReadFile(credPath)
	if err != nil {
		return fmt.Errorf("read credentials: %w", err)
	}

	config, err := google.ConfigFromJSON(credData, drive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\nOpen this URL in your browser:\n%s\n\nPaste the authorization code: ", authURL)

	var authCode string
	fmt.Scanln(&authCode)

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return fmt.Errorf("exchange token: %w", err)
	}

	tokenData, err := json.Marshal(token)
	if err != nil {
		return err
	}

	encPath := filepath.Join(configDir, "google-credentials.json.age")
	if err := crypto.EncryptToFile(encPath, tokenData, identity.Recipient()); err != nil {
		return fmt.Errorf("save encrypted credentials: %w", err)
	}

	credEncPath := filepath.Join(configDir, "google-client.json.age")
	if err := crypto.EncryptToFile(credEncPath, credData, identity.Recipient()); err != nil {
		return fmt.Errorf("save encrypted client config: %w", err)
	}

	fmt.Println("\nDrive setup complete! Credentials stored encrypted.")
	return nil
}

func Sync(notesDir, configDir string, identity *age.HybridIdentity) error {
	ds, err := newDriveSync(notesDir, configDir, identity)
	if err != nil {
		return err
	}
	return ds.syncAll()
}

func newDriveSync(notesDir, configDir string, identity *age.HybridIdentity) (*DriveSync, error) {
	credData, err := crypto.DecryptFile(filepath.Join(configDir, "google-client.json.age"), identity)
	if err != nil {
		return nil, fmt.Errorf("load client config: %w (run 'pq-notes drive setup' first)", err)
	}

	config, err := google.ConfigFromJSON(credData, drive.DriveFileScope)
	if err != nil {
		return nil, err
	}

	tokenData, err := crypto.DecryptFile(filepath.Join(configDir, "google-credentials.json.age"), identity)
	if err != nil {
		return nil, fmt.Errorf("load token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(tokenData, &token); err != nil {
		return nil, err
	}

	client := config.Client(context.Background(), &token)
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &DriveSync{
		service:  srv,
		identity: identity,
		notesDir: notesDir,
		configDir: configDir,
	}, nil
}

func (ds *DriveSync) syncAll() error {
	fmt.Println("Syncing notes to Google Drive...")

	err := filepath.Walk(ds.notesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if info.IsDir() {
			if info.Name() == ".pq-notes" { return filepath.SkipDir }
			return nil
		}
		if filepath.Ext(path) != ".age" { return nil }

		relPath, _ := filepath.Rel(ds.notesDir, path)
		fmt.Printf("  Uploading: %s\n", relPath)
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println("Sync complete!")
	return nil
}
```

- [ ] **Step 2: Create cmd/drive.go**

```go
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/kborup-redhat/pq-notes/internal/config"
	"github.com/kborup-redhat/pq-notes/internal/crypto"
	"github.com/kborup-redhat/pq-notes/internal/drive"
)

var driveCmd = &cobra.Command{
	Use:   "drive",
	Short: "Google Drive sync",
}

var driveSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up Google Drive OAuth2 authentication",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return err
		}
		return drive.Setup(configDir, identity)
	},
}

var driveSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manually sync notes to Google Drive",
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir()
		configDir := config.ConfigDirIn(notesDir)
		identity, err := crypto.LoadIdentity(filepath.Join(configDir, "key.txt"))
		if err != nil {
			return err
		}
		return drive.Sync(notesDir, configDir, identity)
	},
}

var driveAutoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Toggle automatic sync after edits",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := config.ConfigDirIn(config.NotesDir())
		cfg, err := config.Load(configDir)
		if err != nil {
			return err
		}
		cfg.DriveAutoSync = !cfg.DriveAutoSync
		if err := config.Save(cfg, configDir); err != nil {
			return err
		}
		if cfg.DriveAutoSync {
			fmt.Println("Auto-sync enabled")
		} else {
			fmt.Println("Auto-sync disabled")
		}
		return nil
	},
}

func init() {
	driveCmd.AddCommand(driveSetupCmd, driveSyncCmd, driveAutoCmd)
	rootCmd.AddCommand(driveCmd)
}
```

- [ ] **Step 3: Add Google API dependencies and build**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go get google.golang.org/api/drive/v3@latest
go get golang.org/x/oauth2@latest
go build -o pq-notes .
./pq-notes drive --help
```
Expected: builds successfully, drive help text shown

- [ ] **Step 4: Commit**

```bash
git add internal/drive/ cmd/drive.go go.mod go.sum
git commit -m "feat: add Google Drive sync framework with OAuth2 setup and encrypted credential storage"
```

---

### Task 20: Integration Test & Final Build

**Files:**
- Modify: `internal/notes/store_test.go` (add integration test)

- [ ] **Step 1: Write an end-to-end integration test**

Add to `internal/notes/store_test.go`:

```go
func TestIntegrationFullWorkflow(t *testing.T) {
	store, _ := setupTestStore(t)

	// Create notes of different types
	meeting := &Note{
		Customer:  "Acme Corp",
		Type:      Meeting,
		Created:   time.Now(),
		Due:       time.Now().Add(24 * time.Hour),
		Title:     "Sprint Planning",
		Tags:      []string{"sprint", "planning"},
		Status:    StatusOpen,
		Attendees: []string{"Kim", "Sarah"},
	}
	task := &Note{
		Customer: "Red Hat",
		Type:     Task,
		Created:  time.Now(),
		Due:      time.Now().Add(72 * time.Hour),
		Title:    "Fix Login Bug",
		Tags:     []string{"bug", "urgent"},
		Status:   StatusOpen,
		Priority: PriorityUrgent,
	}
	reminder := &Note{
		Customer: "Internal",
		Type:     Reminder,
		Created:  time.Now(),
		Due:      time.Now().Add(1 * time.Hour),
		Title:    "Submit Timecards",
		Status:   StatusOpen,
		Repeat:   "every 2nd-last workday",
	}

	for _, n := range []*Note{meeting, task, reminder} {
		if _, err := store.Create(n); err != nil {
			t.Fatalf("create %s: %v", n.Title, err)
		}
	}

	// List all
	allNotes, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(allNotes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(allNotes))
	}

	// Search
	results, err := store.Search("bug")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Fix Login Bug" {
		t.Errorf("search for 'bug' failed: %v", results)
	}

	// Update status
	taskNote := allNotes[0]
	for _, n := range allNotes {
		if n.Title == "Fix Login Bug" {
			taskNote = n
			break
		}
	}
	taskNote.Status = StatusDone
	if err := store.Update(taskNote.FilePath, taskNote); err != nil {
		t.Fatal(err)
	}

	updated, err := store.Read(taskNote.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusDone {
		t.Error("task should be marked done")
	}
}
```

- [ ] **Step 2: Run all tests**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go test ./... -v
```
Expected: all tests across all packages PASS

- [ ] **Step 3: Final build**

Run:
```bash
cd /home/kborup/ai-code/terminal-notes
go build -o pq-notes .
./pq-notes --help
```
Expected: clean build, help shows all subcommands

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat: add integration tests and finalize pq-notes v0.1.0"
```

---

## Dependency Summary

All Go dependencies used in this plan:

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI command framework |
| `gopkg.in/yaml.v3` | YAML config and frontmatter |
| `filippo.io/age` | Post-quantum encryption |
| `github.com/rickar/cal/v2` | Country holidays and business calendar |
| `charm.land/bubbletea/v2` | TUI framework |
| `charm.land/lipgloss/v2` | Terminal styling |
| `charm.land/glamour/v2` | Markdown rendering |
| `google.golang.org/api/drive/v3` | Google Drive API |
| `golang.org/x/oauth2` | OAuth2 for Drive |
