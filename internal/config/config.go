package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CustomHoliday represents a custom holiday with a name and date in DD-MM format
type CustomHoliday struct {
	Name string `yaml:"name"`
	Date string `yaml:"date"` // DD-MM format
}

// Config represents the application configuration
type Config struct {
	Editor         string          `yaml:"editor"`
	DateFormat     string          `yaml:"date_format"`
	Country        string          `yaml:"country"`
	Weekend        []string        `yaml:"weekend"`
	CustomHolidays []CustomHoliday `yaml:"custom_holidays,omitempty"`
	DriveAutoSync  bool            `yaml:"drive_auto_sync"`
}

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

// Exists checks if the config.yaml file exists in the given config directory
func Exists(configDir string) bool {
	configFile := filepath.Join(configDir, "config.yaml")
	_, err := os.Stat(configFile)
	return err == nil
}

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
