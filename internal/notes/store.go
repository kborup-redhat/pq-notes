package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"filippo.io/age"
	"gopkg.in/yaml.v3"

	"github.com/kborup-redhat/pq-notes/internal/crypto"
)

// NoteStore provides encrypted CRUD operations for notes stored as .md.age files.
type NoteStore struct {
	baseDir    string
	identity   *age.HybridIdentity
	dateFormat string // "eu" or "us"
}

// NewNoteStore creates a new NoteStore.
func NewNoteStore(baseDir string, identity *age.HybridIdentity, dateFormat string) *NoteStore {
	return &NoteStore{
		baseDir:    baseDir,
		identity:   identity,
		dateFormat: dateFormat,
	}
}

// dateLayout converts a "eu" or "us" format string to a Go time layout.
func dateLayout(format string) string {
	if strings.ToUpper(format) == "US" {
		return "01-02-2006 15:04"
	}
	return "02-01-2006 15:04"
}

// createDir creates a directory and all parents if they don't exist.
func createDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}

// Create generates a note template, encrypts it, and writes it to the customer directory.
// Returns the full path of the created file.
func (s *NoteStore) Create(note *Note) (string, error) {
	layout := dateLayout(s.dateFormat)
	content, err := GenerateTemplate(note, layout)
	if err != nil {
		return "", fmt.Errorf("failed to generate template: %w", err)
	}

	customerDir := SanitizeCustomerName(note.Customer)
	dirPath := filepath.Join(s.baseDir, customerDir)
	if err := createDir(dirPath); err != nil {
		return "", fmt.Errorf("failed to create customer directory: %w", err)
	}

	filename := NoteFilename(note.Title, note.Created)
	filePath := filepath.Join(dirPath, filename)

	if err := crypto.EncryptToFile(filePath, []byte(content), s.identity.Recipient()); err != nil {
		return "", fmt.Errorf("failed to encrypt note: %w", err)
	}

	return filePath, nil
}

// Read decrypts and parses a note file. Sets FilePath on the returned Note.
func (s *NoteStore) Read(path string) (*Note, error) {
	layout := dateLayout(s.dateFormat)

	plaintext, err := crypto.DecryptFile(path, s.identity)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt note: %w", err)
	}

	note, err := ParseNote(string(plaintext), layout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse note: %w", err)
	}

	note.FilePath = path
	return note, nil
}

// Update re-encrypts a note to the given path. If the note has a custom Body
// (non-empty), the body is preserved; otherwise a fresh template is generated.
func (s *NoteStore) Update(path string, note *Note) error {
	layout := dateLayout(s.dateFormat)

	var content string
	var err error
	if note.Body != "" {
		content, err = renderWithBody(note, layout)
	} else {
		content, err = GenerateTemplate(note, layout)
	}
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}

	if err := crypto.EncryptToFile(path, []byte(content), s.identity.Recipient()); err != nil {
		return fmt.Errorf("failed to encrypt updated note: %w", err)
	}

	return nil
}

// renderWithBody generates frontmatter and appends the note's existing Body.
func renderWithBody(note *Note, dateFormat string) (string, error) {
	var sb strings.Builder

	fm := frontmatterOut{
		Customer: note.Customer,
		Type:     note.Type.String(),
		Created:  formatDate(note.Created, dateFormat),
		Status:   string(note.Status),
	}

	if !note.Due.IsZero() {
		fm.Due = formatDate(note.Due, dateFormat)
	}
	if note.Repeat != "" {
		fm.Repeat = note.Repeat
	}
	if len(note.Tags) > 0 {
		fm.Tags = note.Tags
	}
	if note.Priority != "" {
		fm.Priority = string(note.Priority)
	}
	if len(note.Attendees) > 0 {
		fm.Attendees = note.Attendees
	}
	if note.Related != "" {
		fm.Related = note.Related
	}

	frontmatterBytes, err := yaml.Marshal(&fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}

	sb.WriteString("---\n")
	sb.Write(frontmatterBytes)
	sb.WriteString("---\n\n")
	sb.WriteString(note.Body)

	return sb.String(), nil
}

// Delete removes a note file from disk.
func (s *NoteStore) Delete(path string) error {
	return os.Remove(path)
}

// List walks the base directory, decrypts and parses all .md.age files,
// skips the .pq-notes directory, and returns notes sorted by Created descending.
func (s *NoteStore) List() ([]*Note, error) {
	var notes []*Note

	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the .pq-notes directory
		if info.IsDir() && info.Name() == ".pq-notes" {
			return filepath.SkipDir
		}

		// Only process .md.age files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md.age") {
			return nil
		}

		note, err := s.Read(path)
		if err != nil {
			// Skip files that can't be decrypted/parsed rather than failing the whole list
			return nil
		}

		notes = append(notes, note)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk notes directory: %w", err)
	}

	// Sort by Created descending (newest first)
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Created.After(notes[j].Created)
	})

	return notes, nil
}

// Search performs a case-insensitive search across title, customer, body, and tags.
// Returns matching notes sorted by Created descending.
func (s *NoteStore) Search(query string) ([]*Note, error) {
	allNotes, err := s.List()
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	var results []*Note

	for _, note := range allNotes {
		if matchesQuery(note, lowerQuery) {
			results = append(results, note)
		}
	}

	return results, nil
}

// matchesQuery checks if a note matches the search query (case-insensitive).
func matchesQuery(note *Note, lowerQuery string) bool {
	if strings.Contains(strings.ToLower(note.Title), lowerQuery) {
		return true
	}
	if strings.Contains(strings.ToLower(note.Customer), lowerQuery) {
		return true
	}
	if strings.Contains(strings.ToLower(note.Body), lowerQuery) {
		return true
	}
	for _, tag := range note.Tags {
		if strings.Contains(strings.ToLower(tag), lowerQuery) {
			return true
		}
	}
	return false
}
