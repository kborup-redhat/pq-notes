---
title: "Chapter 1: Configuration"
order: 1
---

# Chapter 1: Configuration

## Introduction

Think of configuration like the settings panel in your phone. Before you start using any app, it needs to know your preferences: your language, your time zone, your notification style. pq-notes is no different. Before it can create encrypted notes, schedule reminders, or sync to Google Drive, it needs to know a few things about you -- your preferred editor, your date format, your country (for holidays), and which days are your weekend.

The configuration system is the foundation that every other component in pq-notes builds on. The crypto layer reads the config directory to find your key file. The business calendar reads your country and weekend settings. The Drive sync checks whether auto-sync is enabled. Everything starts here.

## How It Works

pq-notes stores its configuration as a YAML file inside a hidden `.pq-notes` directory within your notes folder. The default notes folder is `~/notes/`, making the full config path `~/notes/.pq-notes/config.yaml`. This design keeps all your data -- notes and configuration -- in one directory, which makes backup and sync straightforward.

When you first run `pq-notes`, the setup wizard creates this file. On subsequent launches, the app loads it. If you ever want to change settings, `pq-notes config` reopens the wizard.

Here is what a typical config file looks like:

```yaml
editor: vim
date_format: "02-01-2006"
country: DK
weekend:
  - saturday
  - sunday
drive_auto_sync: true
custom_holidays:
  - name: "Company Founding Day"
    date: "15-06"
```

## Code Deep Dive

The configuration code lives in `internal/config/config.go`. Let's walk through it piece by piece.

### The CustomHoliday Struct

Custom holidays let users define company-specific or personal holidays beyond the national ones. Each holiday has a name and a date in `DD-MM` format (day-month, no year, so it repeats annually):

```go
// CustomHoliday represents a custom holiday with a name and date in DD-MM format
type CustomHoliday struct {
    Name string `yaml:"name"`
    Date string `yaml:"date"` // DD-MM format
}
```

The `yaml` struct tags tell the YAML marshaler how to serialize and deserialize these fields. This is a pattern you will see throughout the codebase -- Go struct tags bridging between Go types and external formats.

Why `DD-MM` and not a full date with a year? Because custom holidays repeat every year. Your company's founding day does not change annually, and neither does a religious observance on a fixed calendar date. By omitting the year, a single entry covers every year automatically. The business calendar (Chapter 5) consumes these entries and converts them into proper `cal.Holiday` objects.

### The Config Struct

The main `Config` struct holds all application settings:

```go
// Config represents the application configuration
type Config struct {
    Editor         string          `yaml:"editor"`
    DateFormat     string          `yaml:"date_format"`
    Country        string          `yaml:"country"`
    Weekend        []string        `yaml:"weekend"`
    CustomHolidays []CustomHoliday `yaml:"custom_holidays,omitempty"`
    DriveAutoSync  bool            `yaml:"drive_auto_sync"`
}
```

A few things to notice:

- **`Editor`** stores the user's preferred text editor (`vim`, `nano`, `code`, etc.). When you edit a note, pq-notes launches this editor.
- **`DateFormat`** uses Go's reference date format (`02-01-2006` for EU, `01-02-2006` for US). Go's time package uses the specific date "Mon Jan 2 15:04:05 MST 2006" as its reference -- the numbers in the format string are not arbitrary.
- **`Country`** is an ISO 3166-1 alpha-2 code (`US`, `DK`, `GB`, etc.) used by the business calendar for national holidays.
- **`Weekend`** is a slice of weekday names. Most countries use Saturday and Sunday, but some use Friday and Saturday.
- **`CustomHolidays`** uses `omitempty` so the YAML output stays clean when there are no custom holidays.
- **`DriveAutoSync`** controls whether notes are automatically synced to Google Drive after every edit.

### Directory Helpers

Two functions establish the directory layout:

```go
// NotesDir returns the default notes directory based on the operating system.
// Returns ~/notes/ on Linux/macOS, %USERPROFILE%\notes\ on Windows
func NotesDir() string {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "notes"
    }
    return filepath.Join(homeDir, "notes")
}

// ConfigDirIn returns the configuration directory within the given notes directory.
// Returns notesDir/.pq-notes
func ConfigDirIn(notesDir string) string {
    return filepath.Join(notesDir, ".pq-notes")
}
```

`NotesDir()` uses `os.UserHomeDir()` to find the user's home directory, then appends `notes`. The fallback to a relative `"notes"` path handles the rare case where the home directory cannot be determined. `ConfigDirIn()` nests the config inside the notes directory, keeping everything together.

This two-function design is intentional. `NotesDir()` returns the default, but users can override it (for example, to store notes on an encrypted volume or a different partition). Whatever directory they choose, `ConfigDirIn()` derives the config location from it. The rest of the codebase always calls these functions rather than hardcoding paths, so changing the notes directory ripples through automatically.

### The File Layout

To put these helpers in context, here is the complete file tree that pq-notes creates:

```
~/notes/                          # NotesDir()
  acme-corp/                      # A note folder
    2026-05-06-q2-review.md.age   # An encrypted note
  personal/
    2026-04-01-grocery-list.md.age
  .pq-notes/                      # ConfigDirIn(notesDir)
    config.yaml                   # This chapter
    key.txt                       # Chapter 2: private key
    contacts.json.age             # Encrypted contacts store
    google-client.json.age        # Encrypted OAuth2 client config
    google-credentials.json.age   # Encrypted OAuth2 tokens
```

Everything under `.pq-notes/` is either configuration or security material. Everything outside it is note data. This clean separation means you can back up `~/notes/` as a single unit and get both your notes and your config.

### Checking for an Existing Config

Before running the setup wizard, pq-notes checks whether a config already exists:

```go
// Exists checks if the config.yaml file exists in the given config directory
func Exists(configDir string) bool {
    configFile := filepath.Join(configDir, "config.yaml")
    _, err := os.Stat(configFile)
    return err == nil
}
```

This uses `os.Stat` -- a standard Go pattern for checking file existence. If `Stat` returns no error, the file exists. Note that the function only checks for `config.yaml`, not for the directory itself. This means a partially created config directory (perhaps from a crashed first run) will still trigger the setup wizard, which is the safer behavior.

### Saving Configuration

The `Save` function marshals the config to YAML and writes it to disk:

```go
// Save marshals the config to YAML and writes it to config.yaml in the config directory.
// Creates the config directory if it doesn't exist.
func Save(cfg *Config, configDir string) error {
    // Create config directory if it doesn't exist
    err := os.MkdirAll(configDir, 0700)
    if err != nil {
        return err
    }

    // Marshal config to YAML
    data, err := yaml.Marshal(cfg)
    if err != nil {
        return err
    }

    // Write to config.yaml
    configFile := filepath.Join(configDir, "config.yaml")
    return os.WriteFile(configFile, data, 0600)
}
```

Notice the file permissions: `0700` for the directory (owner-only access) and `0600` for the file (owner read/write only). This is a security-conscious choice -- the config directory also holds the private encryption key, so restricting access matters.

`os.MkdirAll` creates the directory and any missing parents, making `Save` safe to call even on first run.

### Loading Configuration

Loading is the reverse operation:

```go
// Load reads and unmarshals the config.yaml from the config directory
func Load(configDir string) (*Config, error) {
    configFile := filepath.Join(configDir, "config.yaml")
    data, err := os.ReadFile(configFile)
    if err != nil {
        return nil, err
    }

    var cfg Config
    err = yaml.Unmarshal(data, &cfg)
    if err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

Read the file, unmarshal it into a `Config` struct, return the pointer. Straightforward Go error handling: every operation that can fail returns an error, and we propagate it up.

One thing to note: `Load` returns a pointer to the config (`*Config`), not a value copy. This means the caller gets a reference to the actual struct, which is efficient (no copying) and allows the caller to modify it in memory (for example, during the setup wizard) before passing it back to `Save`. This pointer-based approach is idiomatic Go for structs that represent configuration or state.

### Country-Aware Weekend Defaults

Not every country has a Saturday-Sunday weekend. The `DefaultWeekend` function handles this:

```go
// DefaultWeekend returns the default weekend days for a given country.
// Returns ["friday","saturday"] for SA/AE/BH/KW/OM/QA/IL, else ["saturday","sunday"]
func DefaultWeekend(country string) []string {
    // Countries with Friday-Saturday weekend
    fridaySaturdayCountries := map[string]bool{
        "SA": true, // Saudi Arabia
        "AE": true, // United Arab Emirates
        "BH": true, // Bahrain
        "KW": true, // Kuwait
        "OM": true, // Oman
        "QA": true, // Qatar
        "IL": true, // Israel
    }

    if fridaySaturdayCountries[country] {
        return []string{"friday", "saturday"}
    }

    return []string{"saturday", "sunday"}
}
```

This is used during setup to pre-populate the weekend field. The setup wizard calls `DefaultWeekend` with the user's selected country code and fills in the appropriate defaults. The user can still override this -- the config stores whatever they choose.

The map-based lookup is a clean pattern: rather than a chain of `if` statements, a `map[string]bool` provides O(1) lookup and makes it easy to add or remove countries.

This function is only used for setting defaults during initial setup. Once the user has a config file, their actual weekend preference comes from the `Weekend` field in the config. If a user in the US wants a three-day weekend, they can set `weekend: ["friday", "saturday", "sunday"]` and the business calendar will respect it.

## Relationships to Other Components

The configuration system is referenced throughout the codebase:

- **Crypto (Chapter 2):** The key file is stored in the config directory at `<configDir>/key.txt`. `GenerateKey` and `LoadIdentity` both receive the config directory path to locate the key file.
- **Note Model (Chapter 3):** The `DateFormat` field from config is passed to `ParseNote` and `GenerateTemplate` so dates in frontmatter match the user's locale preference.
- **Date Utilities (Chapter 4):** `FormatDate` and `ParseDate` accept a format string (`"EU"` or `"US"`) that corresponds to the config's `DateFormat`.
- **Business Calendar (Chapter 5):** The `New()` constructor takes a `*Config` directly, reading `Country`, `Weekend`, and `CustomHolidays` to build a fully configured business calendar.
- **Drive Sync:** Checks `DriveAutoSync` to decide whether to upload after every note edit.

## Key Takeaways

- Configuration is stored as YAML in `~/notes/.pq-notes/config.yaml`, keeping all data in one directory tree.
- File permissions are locked down (`0700` for directories, `0600` for files) because the config directory also holds the private key.
- `DefaultWeekend()` maps ISO country codes to the correct weekend days, supporting 7 countries with Friday-Saturday weekends.
- The `omitempty` YAML tag keeps serialized output clean by omitting empty optional fields.
- Go's `os.UserHomeDir()` and `filepath.Join()` ensure cross-platform path handling.

## Next Steps

Now that we know how pq-notes stores its settings, let's look at the most critical component that lives alongside the config: the encryption system. In [Chapter 2: Post-Quantum Cryptography](02-Cryptography.md), we will explore how pq-notes generates hybrid keys, encrypts notes, and why post-quantum security matters today.
