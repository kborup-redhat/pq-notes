---
title: "Chapter 7: NoteStore -- Encrypted CRUD"
order: 7
---

# Chapter 7: NoteStore -- Encrypted CRUD

Every application that manages data needs a storage layer. For pq-notes, that layer has an unusual constraint: every note must be encrypted at rest using post-quantum cryptography. The NoteStore is the gatekeeper -- it handles creating, reading, updating, deleting, listing, and searching notes, and it ensures that plaintext never touches the disk.

Think of NoteStore as a filing clerk who works in a locked room. You hand the clerk a note, and the clerk puts it in a sealed envelope before filing it. When you ask for a note back, the clerk opens the envelope, hands you the contents, and re-seals it when you are done. The clerk never lets an unsealed note leave the room.

## How It Works

NoteStore sits between the application logic (TUI, CLI) and the filesystem. Every operation flows through encryption:

1. **Create** -- generate a note template, encrypt it, write the `.md.age` file
2. **Read** -- decrypt the file, parse the frontmatter and body into a `Note` struct
3. **Update** -- re-render the note content, re-encrypt, overwrite the file
4. **Delete** -- remove the file from disk
5. **List** -- walk the directory tree, decrypt and parse every `.md.age` file, sort by date
6. **Search** -- list all notes, then filter by a case-insensitive query across multiple fields

The store organizes notes into folders on disk. Each folder becomes a subdirectory under the base notes directory:

```
~/notes/
    .pq-notes/          # config directory (skipped during listing)
    acme-corp/
        2026-05-06-q2-budget-review.md.age
    personal/
        2026-05-01-grocery-list.md.age
```

## Code Deep Dive

### The NoteStore Struct

The struct holds three pieces of state:

```go
type NoteStore struct {
    baseDir    string
    identity   *age.HybridIdentity
    dateFormat string // "eu" or "us"
}
```

- **`baseDir`** -- the root directory where all notes live (e.g., `~/notes`)
- **`identity`** -- the user's age hybrid identity, used for both encryption (via its recipient/public key) and decryption (via its private key)
- **`dateFormat`** -- controls how dates are formatted in frontmatter (`"eu"` for DD-MM-YYYY, `"us"` for MM-DD-YYYY)

The constructor is straightforward:

```go
func NewNoteStore(baseDir string, identity *age.HybridIdentity, dateFormat string) *NoteStore {
    return &NoteStore{
        baseDir:    baseDir,
        identity:   identity,
        dateFormat: dateFormat,
    }
}
```

### The dateLayout Helper

Go's `time` package uses a reference date (`01/02 03:04:05PM '06`) to define formats. The `dateLayout` helper converts the user's preference into a Go time layout string:

```go
func dateLayout(format string) string {
    if strings.ToUpper(format) == "US" {
        return "01-02-2006 15:04"
    }
    return "02-01-2006 15:04"
}
```

The US format puts month first (`01-02-2006`), while the EU format puts day first (`02-01-2006`). This helper is called at the start of every CRUD operation to ensure consistent date handling.

### Create -- Writing a New Note

Creating a note involves four steps: generate the template, create the folder directory, encrypt the content, and write the file:

```go
func (s *NoteStore) Create(note *Note) (string, error) {
    layout := dateLayout(s.dateFormat)
    content, err := GenerateTemplate(note, layout)
    if err != nil {
        return "", fmt.Errorf("failed to generate template: %w", err)
    }

    folderDir := SanitizeFolderName(note.Folder)
    dirPath := filepath.Join(s.baseDir, folderDir)
    if err := createDir(dirPath); err != nil {
        return "", fmt.Errorf("failed to create folder directory: %w", err)
    }

    filename := NoteFilename(note.Title, note.Created)
    filePath := filepath.Join(dirPath, filename)

    if err := crypto.EncryptToFile(filePath, []byte(content), s.identity.Recipient()); err != nil {
        return "", fmt.Errorf("failed to encrypt note: %w", err)
    }

    return filePath, nil
}
```

The `createDir` helper uses `os.MkdirAll` with `0700` permissions (owner-only access):

```go
func createDir(dir string) error {
    return os.MkdirAll(dir, 0700)
}
```

Notice the permission difference from the config directory (`0755`). Notes directories use `0700` because they contain encrypted personal data -- even though the files themselves are encrypted, there is no reason to let other users on the system browse the directory structure and see folder names.

The function returns the full file path so the caller can immediately open the note in the editor.

### Read -- Decrypting and Parsing

Reading reverses the creation process: decrypt first, then parse:

```go
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
```

A subtle but important detail: `note.FilePath = path` sets the file path on the parsed note. The `FilePath` field is tagged `yaml:"-"` in the Note struct, meaning it does not exist in the serialized content. The store injects it after parsing so that downstream code (like the TUI) knows where each note lives on disk without having to track paths separately.

### Update -- Re-encrypting with Body Preservation

Updating is more nuanced than creating. When the user edits a note in their text editor, the body content changes. The Update function needs to preserve that body while regenerating the frontmatter:

```go
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
```

The branch logic is important: if the note has a body (meaning the user has written content), use `renderWithBody` to preserve it. If the body is empty (perhaps a metadata-only update like toggling status), regenerate a fresh template.

The `renderWithBody` function builds frontmatter from the note's fields and appends the existing body:

```go
func renderWithBody(note *Note, dateFormat string) (string, error) {
    var sb strings.Builder

    fm := frontmatterOut{
        Folder:  note.Folder,
        Type:    note.Type.String(),
        Created: formatDate(note.Created, dateFormat),
        Status:  string(note.Status),
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
```

This function carefully checks each optional field before including it. Zero-value fields (empty strings, nil slices, zero times) are omitted from the frontmatter to keep the YAML clean. The body is appended verbatim after the closing `---` delimiter.

### Delete

Delete is the simplest operation -- just remove the file:

```go
func (s *NoteStore) Delete(path string) error {
    return os.Remove(path)
}
```

No decryption is needed. The function does not clean up empty parent directories; if a folder's last note is deleted, the empty folder remains. This is intentional -- the folder might be reused soon, and empty directories are harmless.

### List -- Walking the Directory Tree

Listing notes means walking the entire directory tree, decrypting every `.md.age` file, and sorting the results:

```go
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
            // Skip files that can't be decrypted/parsed rather than failing
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
```

There are several design decisions worth calling out:

- **`.pq-notes` is skipped** using `filepath.SkipDir`, which tells `Walk` to skip the entire subtree. This prevents config files from being treated as notes.
- **Non-`.md.age` files are ignored** silently, so you can put other files (like a README) in the notes directory without breaking anything.
- **Decryption failures are swallowed** -- if a file cannot be decrypted or parsed, it is skipped rather than failing the entire listing. This is a resilience choice: one corrupted file should not prevent you from seeing all your other notes.
- **Sorting is newest-first**, which is the natural expectation in a note-taking app. The sort uses `time.After` for descending order.

### Search -- Full-Text Across Fields

Search builds on top of List, adding a filter:

```go
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
```

The `matchesQuery` helper checks four fields:

```go
func matchesQuery(note *Note, lowerQuery string) bool {
    if strings.Contains(strings.ToLower(note.Title), lowerQuery) {
        return true
    }
    if strings.Contains(strings.ToLower(note.Folder), lowerQuery) {
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
```

The search is case-insensitive (both query and field values are lowercased) and checks title, folder name, body content, and tags. It returns results in the same newest-first order as `List` because it filters the already-sorted output.

This is a linear scan -- every note is decrypted and checked. For a personal note-taking app with hundreds or even a few thousand notes, this is perfectly adequate. An index would add complexity and create an unencrypted metadata leak.

## Relationships

- **Crypto** (`internal/crypto`) provides `EncryptToFile` and `DecryptFile`. NoteStore never handles raw encryption -- it delegates to the crypto layer.
- **Note Model** (`internal/notes`) provides `ParseNote`, `GenerateTemplate`, `NoteFilename`, and `SanitizeFolderName`. NoteStore orchestrates these building blocks.
- **Editor** (`internal/editor`) works with NoteStore indirectly: the TUI creates a note via NoteStore, gets a file path, and hands that path to the editor for encrypted editing.
- **Config** provides the `dateFormat` setting that NoteStore passes through to date formatting functions.
- **TUI** calls `List`, `Search`, `Create`, `Update`, and `Delete` to drive the user interface.

## Key Takeaways

- **Encryption is invisible to callers** -- NoteStore's API looks like any other CRUD layer. The encryption/decryption is an implementation detail hidden behind `Create`, `Read`, `Update`, `Delete`.
- **`filepath.Walk` with `SkipDir`** is Go's standard way to traverse directories while excluding specific subtrees.
- **Graceful degradation** (skipping unreadable files in `List`) keeps the application usable even if some data is corrupted.
- **Body preservation** via `renderWithBody` ensures that updating metadata (like toggling status) does not destroy the user's written content.
- **Linear search is acceptable** when the dataset is small and the alternative (an index) would leak metadata.

## Next Steps

We can now create, read, update, and delete encrypted notes. But how does the user actually write content? In the next chapter, we will build the Editor Integration layer that launches external text editors, handles the decrypt-edit-re-encrypt cycle, and securely cleans up temporary files.
