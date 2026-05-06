---
title: "Chapter 1: Configuration"
order: 1
---

# Chapter 1: Configuration

Every application needs a way to remember its settings. For pq-notes, that means knowing which text editor to use, how to format dates, which country's holidays to observe, and more. In this chapter, we'll build a configuration system that stores settings in a YAML file and provides sensible defaults based on the user's locale.

Think of the Config package as the application's memory — it remembers the user's preferences so they don't have to specify them every time.

## How It Works

The configuration system has four main responsibilities:

1. **Define the shape of settings** — what can be configured
2. **Determine where files live** — OS-aware directory paths
3. **Persist settings** — save to and load from YAML
4. **Provide defaults** — country-specific weekend days

All configuration is stored in a single `config.yaml` file inside a `.pq-notes` hidden directory within the user's notes folder.

## Code Deep Dive

### The Config Struct

The heart of the package is the `Config` struct, which maps directly to a YAML file:

```go
type CustomHoliday struct {
    Name string `yaml:"name"`
    Date string `yaml:"date"` // DD-MM format
}

type Config struct {
    Editor         string          `yaml:"editor"`
    DateFormat     string          `yaml:"date_format"`
    Country        string          `yaml:"country"`
    Weekend        []string        `yaml:"weekend"`
    CustomHolidays []CustomHoliday `yaml:"custom_holidays,omitempty"`
    DriveAutoSync  bool            `yaml:"drive_auto_sync"`
}
```

Each field has a `yaml` struct tag that controls how it serializes. The `omitempty` tag on `CustomHolidays` means the field is omitted from the YAML output when the slice is empty, keeping the config file clean.

A typical `config.yaml` looks like this:

```yaml
editor: code
date_format: EU
country: DK
weekend:
  - saturday
  - sunday
drive_auto_sync: false
```

### OS-Aware Paths

The `NotesDir` function determines where notes are stored based on the operating system:

```go
func NotesDir() string {
    if runtime.GOOS == "windows" {
        userProfile := os.Getenv("USERPROFILE")
        return filepath.Join(userProfile, "notes")
    }

    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "notes"
    }
    return filepath.Join(homeDir, "notes")
}
```

On Linux and macOS, notes go in `~/notes/`. On Windows, they go in `%USERPROFILE%\notes\`. The `filepath.Join` function handles path separator differences automatically — forward slashes on Unix, backslashes on Windows.

If the home directory can't be determined (rare, but possible in containerized environments), the function falls back to a relative `notes` directory.

### Configuration Directory

The configuration directory sits inside the notes directory as a hidden folder:

```go
func ConfigDirIn(notesDir string) string {
    return filepath.Join(notesDir, ".pq-notes")
}
```

This keeps configuration co-located with the notes it applies to. If you move your notes folder (e.g., to a different drive or sync service), the configuration travels with it.

### Checking for Existing Config

Before prompting a user through first-time setup, the application checks whether a config file already exists:

```go
func Exists(configDir string) bool {
    configFile := filepath.Join(configDir, "config.yaml")
    _, err := os.Stat(configFile)
    return err == nil
}
```

This uses `os.Stat` to check file existence. The pattern of discarding the `FileInfo` return value (`_`) and only checking for an error is idiomatic Go for existence checks.

### Saving Configuration

The `Save` function handles both creating the config directory and writing the YAML file:

```go
func Save(cfg *Config, configDir string) error {
    err := os.MkdirAll(configDir, 0755)
    if err != nil {
        return err
    }

    data, err := yaml.Marshal(cfg)
    if err != nil {
        return err
    }

    configFile := filepath.Join(configDir, "config.yaml")
    return os.WriteFile(configFile, data, 0644)
}
```

Key details:

- **`os.MkdirAll`** creates the directory and all parent directories if they don't exist. It's safe to call even if the directory already exists.
- **`0755`** permissions mean the owner can read/write/execute, and others can read/execute (standard for directories).
- **`0644`** permissions for the file mean the owner can read/write, others can only read (standard for non-sensitive config files).
- The function takes `*Config` (a pointer) to avoid copying the struct, and returns an error for the caller to handle.

### Loading Configuration

Loading reverses the process — read the file, then unmarshal the YAML:

```go
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

Notice that `Load` returns `(*Config, error)` — the Go convention for functions that can fail. The caller gets either a valid config pointer or an error, never both.

### Country-Specific Defaults

Different countries have different weekend days. Most of the world observes Saturday-Sunday weekends, but several Middle Eastern and North African countries use Friday-Saturday:

```go
func DefaultWeekend(country string) []string {
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

This function uses a map as a set (the values are all `true`) for O(1) lookup. The country codes follow the ISO 3166-1 alpha-2 standard, the same format used in the config file.

## Relationships

- **NoteStore** reads the `DateFormat` from Config to determine how dates are serialized in note frontmatter.
- **BusinessCal** uses `Country`, `Weekend`, and `CustomHolidays` to configure its holiday and workday calculations.
- **Editor** uses the `Editor` field to know which program to launch.

The Config package has no dependencies on other internal packages — it's a foundational layer that other packages build on.

## Key Takeaways

- **Use struct tags** (`yaml:"..."`) to control serialization format and field names.
- **`os.MkdirAll`** is idempotent — safe to call repeatedly without checking if the directory exists first.
- **`filepath.Join`** handles OS-specific path separators, making code portable across platforms.
- **Return `(value, error)` pairs** from functions that can fail — this is Go's primary error-handling pattern.
- **Co-locate configuration with data** so settings travel with the files they describe.

## Next Steps

With configuration in place, we know *how* the user wants their notes handled. In the next chapter, we'll build the encryption layer that keeps those notes secure — even against future quantum computers.
