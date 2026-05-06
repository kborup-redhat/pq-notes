package notes

import (
	"path/filepath"
	"strings"
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
		Folder: "Acme Corp",
		Type:     Meeting,
		Created:  time.Date(2026, 5, 6, 14, 30, 0, 0, time.UTC),
		Status:   StatusOpen,
		Priority: PriorityHigh,
		Title:    "Q2 Planning",
		Tags:     []string{"planning"},
	}

	path, err := store.Create(note)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Check that the file was created in the correct folder directory
	expectedDir := filepath.Join(tmpDir, "acme-corp")
	if !strings.HasPrefix(path, expectedDir) {
		t.Errorf("path %q should be under folder dir %q", path, expectedDir)
	}

	// Check the file has .md.age extension
	if !strings.HasSuffix(path, ".md.age") {
		t.Errorf("path %q should end with .md.age", path)
	}

	// Verify the file actually exists by trying to read it back
	_, err = store.Read(path)
	if err != nil {
		t.Fatalf("failed to read back created note: %v", err)
	}
}

func TestStoreListAndRead(t *testing.T) {
	store, _ := setupTestStore(t)

	note := &Note{
		Folder: "Acme Corp",
		Type:     Task,
		Created:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		Status:   StatusOpen,
		Priority: PriorityNormal,
		Title:    "Implement feature X",
		Tags:     []string{"dev"},
	}

	path, err := store.Create(note)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// List should return the created note
	notes, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(notes) != 1 {
		t.Fatalf("List() returned %d notes, want 1", len(notes))
	}

	listed := notes[0]
	if listed.Folder != "Acme Corp" {
		t.Errorf("Folder = %v, want Acme Corp", listed.Folder)
	}
	if listed.Title != "Implement feature X" {
		t.Errorf("Title = %v, want Implement feature X", listed.Title)
	}
	if listed.Type != Task {
		t.Errorf("Type = %v, want %v", listed.Type, Task)
	}
	if listed.FilePath == "" {
		t.Error("FilePath should be set on listed note")
	}

	// Read should return the same note with FilePath set
	readNote, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if readNote.FilePath != path {
		t.Errorf("FilePath = %v, want %v", readNote.FilePath, path)
	}
	if readNote.Title != "Implement feature X" {
		t.Errorf("Title = %v, want Implement feature X", readNote.Title)
	}
	if readNote.Status != StatusOpen {
		t.Errorf("Status = %v, want %v", readNote.Status, StatusOpen)
	}
}

func TestStoreUpdate(t *testing.T) {
	store, _ := setupTestStore(t)

	note := &Note{
		Folder: "Test Folder",
		Type:     Task,
		Created:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		Status:   StatusOpen,
		Priority: PriorityNormal,
		Title:    "Update test task",
	}

	path, err := store.Create(note)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Read it back
	readNote, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	// Modify the status
	readNote.Status = StatusDone

	// Update with custom body preserved
	err = store.Update(path, readNote)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Read back and verify the update took effect
	updated, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read() after Update error = %v", err)
	}

	if updated.Status != StatusDone {
		t.Errorf("Status = %v, want %v", updated.Status, StatusDone)
	}
	if updated.Title != "Update test task" {
		t.Errorf("Title = %v, want Update test task", updated.Title)
	}
	// Body should be preserved from the original read
	if updated.Body == "" {
		t.Error("Body should be preserved after update")
	}
}

func TestStoreUpdate_PreservesCustomBody(t *testing.T) {
	store, _ := setupTestStore(t)

	note := &Note{
		Folder: "Test Folder",
		Type:     Meeting,
		Created:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		Status:   StatusOpen,
		Title:    "Body preservation test",
	}

	path, err := store.Create(note)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Read, set custom body content, update
	readNote, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	customBody := "# Body preservation test\n\n## Agenda\nCustom agenda content here\n\n## Notes\nDetailed meeting notes\n\n## Action Items\n- Follow up with team"
	readNote.Body = customBody

	err = store.Update(path, readNote)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Read back and verify body was preserved
	updated, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read() after Update error = %v", err)
	}

	if !strings.Contains(updated.Body, "Custom agenda content here") {
		t.Errorf("Body should contain custom content, got: %v", updated.Body)
	}
	if !strings.Contains(updated.Body, "Detailed meeting notes") {
		t.Errorf("Body should contain detailed notes, got: %v", updated.Body)
	}
}

func TestStoreSearch(t *testing.T) {
	store, _ := setupTestStore(t)

	// Create 3 notes with different content
	notes := []*Note{
		{
			Folder: "Acme Corp",
			Type:     Meeting,
			Created:  time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),
			Status:   StatusOpen,
			Title:    "Q2 Planning Session",
			Tags:     []string{"planning", "quarterly"},
		},
		{
			Folder: "Beta Inc",
			Type:     Task,
			Created:  time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),
			Status:   StatusOpen,
			Title:    "Fix login bug",
			Tags:     []string{"bug", "urgent"},
		},
		{
			Folder: "Acme Corp",
			Type:     Followup,
			Created:  time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC),
			Status:   StatusDone,
			Title:    "Follow up on contract",
			Tags:     []string{"contract"},
		},
	}

	for _, n := range notes {
		_, err := store.Create(n)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Search by title (case-insensitive)
	results, err := store.Search("planning")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search('planning') returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Title != "Q2 Planning Session" {
		t.Errorf("Search result Title = %v, want Q2 Planning Session", results[0].Title)
	}

	// Search by folder name
	results, err = store.Search("acme")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search('acme') returned %d results, want 2", len(results))
	}

	// Search by tag
	results, err = store.Search("urgent")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search('urgent') returned %d results, want 1", len(results))
	}

	// Search with no results
	results, err = store.Search("nonexistent")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search('nonexistent') returned %d results, want 0", len(results))
	}

	// Search case-insensitive
	results, err = store.Search("PLANNING")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search('PLANNING') returned %d results, want 1", len(results))
	}
}

func TestStoreList_SkipsPqNotesDir(t *testing.T) {
	store, tmpDir := setupTestStore(t)

	// Create a note normally
	note := &Note{
		Folder: "Test Folder",
		Type:     Reminder,
		Created:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		Status:   StatusOpen,
		Title:    "Test reminder",
	}
	_, err := store.Create(note)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create a .pq-notes directory with a fake .md.age file
	pqDir := filepath.Join(tmpDir, ".pq-notes")
	err = createDir(pqDir)
	if err != nil {
		t.Fatalf("failed to create .pq-notes dir: %v", err)
	}
	err = crypto.EncryptToFile(
		filepath.Join(pqDir, "config.md.age"),
		[]byte("fake content"),
		store.identity.Recipient(),
	)
	if err != nil {
		t.Fatalf("failed to create fake file in .pq-notes: %v", err)
	}

	// List should only return the real note, not the .pq-notes file
	notes, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("List() returned %d notes, want 1 (should skip .pq-notes)", len(notes))
	}
}

func TestStoreList_SortsByCreatedDesc(t *testing.T) {
	store, _ := setupTestStore(t)

	// Create notes with different creation times
	times := []time.Time{
		time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC),  // oldest
		time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),  // newest
		time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),  // middle
	}

	for i, created := range times {
		note := &Note{
			Folder: "Test Folder",
			Type:     Task,
			Created:  created,
			Status:   StatusOpen,
			Title:    "Note " + string(rune('A'+i)),
		}
		_, err := store.Create(note)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	notes, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(notes) != 3 {
		t.Fatalf("List() returned %d notes, want 3", len(notes))
	}

	// Should be sorted by Created descending (newest first)
	if notes[0].Created.Before(notes[1].Created) {
		t.Errorf("notes[0].Created (%v) should be after notes[1].Created (%v)", notes[0].Created, notes[1].Created)
	}
	if notes[1].Created.Before(notes[2].Created) {
		t.Errorf("notes[1].Created (%v) should be after notes[2].Created (%v)", notes[1].Created, notes[2].Created)
	}
}

func TestIntegrationFullWorkflow(t *testing.T) {
	store, _ := setupTestStore(t)

	meeting := &Note{
		Folder:  "Acme Corp",
		Type:      Meeting,
		Created:   time.Now(),
		Due:       time.Now().Add(24 * time.Hour),
		Title:     "Sprint Planning",
		Tags:      []string{"sprint", "planning"},
		Status:    StatusOpen,
		Attendees: []string{"Kim", "Sarah"},
	}
	task := &Note{
		Folder: "Red Hat",
		Type:     Task,
		Created:  time.Now(),
		Due:      time.Now().Add(72 * time.Hour),
		Title:    "Fix Login Bug",
		Tags:     []string{"bug", "urgent"},
		Status:   StatusOpen,
		Priority: PriorityUrgent,
	}
	reminder := &Note{
		Folder: "Internal",
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

	allNotes, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(allNotes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(allNotes))
	}

	results, err := store.Search("bug")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Fix Login Bug" {
		t.Errorf("search for 'bug' failed: got %d results", len(results))
	}

	var taskNote *Note
	for _, n := range allNotes {
		if n.Title == "Fix Login Bug" {
			taskNote = n
			break
		}
	}
	if taskNote == nil {
		t.Fatal("could not find Fix Login Bug note")
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

func TestDateLayout(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"eu", "02-01-2006 15:04"},
		{"EU", "02-01-2006 15:04"},
		{"us", "01-02-2006 15:04"},
		{"US", "01-02-2006 15:04"},
		{"anything", "02-01-2006 15:04"}, // defaults to EU
		{"", "02-01-2006 15:04"},         // empty defaults to EU
	}

	for _, tt := range tests {
		t.Run("format="+tt.format, func(t *testing.T) {
			got := dateLayout(tt.format)
			if got != tt.want {
				t.Errorf("dateLayout(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}
