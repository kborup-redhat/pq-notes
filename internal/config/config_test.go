package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNotesDir(t *testing.T) {
	notesDir := NotesDir()

	if runtime.GOOS == "windows" {
		userProfile := os.Getenv("USERPROFILE")
		expected := filepath.Join(userProfile, "notes")
		if notesDir != expected {
			t.Errorf("NotesDir() on Windows = %v, want %v", notesDir, expected)
		}
	} else {
		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, "notes")
		if notesDir != expected {
			t.Errorf("NotesDir() on Linux/macOS = %v, want %v", notesDir, expected)
		}
	}
}

func TestConfigDirIn(t *testing.T) {
	tests := []struct {
		name     string
		notesDir string
		want     string
	}{
		{
			name:     "Unix path",
			notesDir: "/home/user/notes",
			want:     "/home/user/notes/.pq-notes",
		},
		{
			name:     "Windows path",
			notesDir: "C:\\Users\\user\\notes",
			want:     filepath.Join("C:\\Users\\user\\notes", ".pq-notes"),
		},
		{
			name:     "Relative path",
			notesDir: "notes",
			want:     filepath.Join("notes", ".pq-notes"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConfigDirIn(tt.notesDir)
			if got != tt.want {
				t.Errorf("ConfigDirIn(%v) = %v, want %v", tt.notesDir, got, tt.want)
			}
		})
	}
}

func TestExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".pq-notes")

	// Test non-existent config directory
	if Exists(configDir) {
		t.Errorf("Exists(%v) = true, want false for non-existent directory", configDir)
	}

	// Create config directory
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Test existent directory but no config.yaml
	if Exists(configDir) {
		t.Errorf("Exists(%v) = true, want false when config.yaml doesn't exist", configDir)
	}

	// Create config.yaml
	configFile := filepath.Join(configDir, "config.yaml")
	err = os.WriteFile(configFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Test existent config.yaml
	if !Exists(configDir) {
		t.Errorf("Exists(%v) = false, want true when config.yaml exists", configDir)
	}
}

func TestDefaultWeekend(t *testing.T) {
	tests := []struct {
		country string
		want    []string
	}{
		{"SA", []string{"friday", "saturday"}},
		{"AE", []string{"friday", "saturday"}},
		{"BH", []string{"friday", "saturday"}},
		{"KW", []string{"friday", "saturday"}},
		{"OM", []string{"friday", "saturday"}},
		{"QA", []string{"friday", "saturday"}},
		{"IL", []string{"friday", "saturday"}},
		{"DK", []string{"saturday", "sunday"}},
		{"US", []string{"saturday", "sunday"}},
		{"GB", []string{"saturday", "sunday"}},
		{"DE", []string{"saturday", "sunday"}},
		{"", []string{"saturday", "sunday"}},
		{"xx", []string{"saturday", "sunday"}},
	}

	for _, tt := range tests {
		t.Run(tt.country, func(t *testing.T) {
			got := DefaultWeekend(tt.country)
			if len(got) != len(tt.want) {
				t.Errorf("DefaultWeekend(%v) length = %v, want %v", tt.country, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("DefaultWeekend(%v)[%d] = %v, want %v", tt.country, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".pq-notes")

	// Create test config
	cfg := &Config{
		Editor:        "vim",
		DateFormat:    "2006-01-02",
		Country:       "DK",
		Weekend:       []string{"saturday", "sunday"},
		DriveAutoSync: true,
		CustomHolidays: []CustomHoliday{
			{Name: "Test Holiday", Date: "01-01"},
			{Name: "Another Holiday", Date: "25-12"},
		},
	}

	// Test Save
	err := Save(cfg, configDir)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify config directory was created
	if !Exists(configDir) {
		t.Errorf("Config directory not created after Save()")
	}

	// Test Load
	loadedCfg, err := Load(configDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded config matches saved config
	if loadedCfg.Editor != cfg.Editor {
		t.Errorf("Load() Editor = %v, want %v", loadedCfg.Editor, cfg.Editor)
	}
	if loadedCfg.DateFormat != cfg.DateFormat {
		t.Errorf("Load() DateFormat = %v, want %v", loadedCfg.DateFormat, cfg.DateFormat)
	}
	if loadedCfg.Country != cfg.Country {
		t.Errorf("Load() Country = %v, want %v", loadedCfg.Country, cfg.Country)
	}
	if loadedCfg.DriveAutoSync != cfg.DriveAutoSync {
		t.Errorf("Load() DriveAutoSync = %v, want %v", loadedCfg.DriveAutoSync, cfg.DriveAutoSync)
	}
	if len(loadedCfg.Weekend) != len(cfg.Weekend) {
		t.Errorf("Load() Weekend length = %v, want %v", len(loadedCfg.Weekend), len(cfg.Weekend))
	} else {
		for i := range loadedCfg.Weekend {
			if loadedCfg.Weekend[i] != cfg.Weekend[i] {
				t.Errorf("Load() Weekend[%d] = %v, want %v", i, loadedCfg.Weekend[i], cfg.Weekend[i])
			}
		}
	}
	if len(loadedCfg.CustomHolidays) != len(cfg.CustomHolidays) {
		t.Errorf("Load() CustomHolidays length = %v, want %v", len(loadedCfg.CustomHolidays), len(cfg.CustomHolidays))
	} else {
		for i := range loadedCfg.CustomHolidays {
			if loadedCfg.CustomHolidays[i].Name != cfg.CustomHolidays[i].Name {
				t.Errorf("Load() CustomHolidays[%d].Name = %v, want %v", i, loadedCfg.CustomHolidays[i].Name, cfg.CustomHolidays[i].Name)
			}
			if loadedCfg.CustomHolidays[i].Date != cfg.CustomHolidays[i].Date {
				t.Errorf("Load() CustomHolidays[%d].Date = %v, want %v", i, loadedCfg.CustomHolidays[i].Date, cfg.CustomHolidays[i].Date)
			}
		}
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".pq-notes-nonexistent")

	_, err := Load(configDir)
	if err == nil {
		t.Errorf("Load() with non-existent config should return error, got nil")
	}
}

func TestSaveCreatesDirs(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "nested", "path", ".pq-notes")

	cfg := &Config{
		Editor:     "nano",
		DateFormat: "02-01-2006",
		Country:    "US",
		Weekend:    []string{"saturday", "sunday"},
	}

	err := Save(cfg, configDir)
	if err != nil {
		t.Fatalf("Save() with nested path error = %v", err)
	}

	if !Exists(configDir) {
		t.Errorf("Save() should create nested directories, but config doesn't exist")
	}
}
