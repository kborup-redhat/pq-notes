---
title: "Chapter 6: NoteStore"
order: 6
---

# Chapter 6: The NoteStore

The NoteStore is where everything comes together. It's the orchestration layer that combines the crypto package, the note model, and the filesystem into a cohesive encrypted CRUD (Create, Read, Update, Delete) interface. If the Note model is the blueprint, the NoteStore is the construction crew.

Think of it as a secure filing cabinet — it knows how to file new notes, retrieve existing ones, and search through the entire collection, all while keeping everything encrypted.

## How It Works

The NoteStore provides five operations:

1. **Create** — generate a template, encrypt it, save to the customer's directory
2. **Read** — decrypt a file, parse it into a Note struct
3. **Update** — re-encrypt modified note content
4. **List** — walk the directory tree, decrypt and parse all notes
5. **Search** — list all notes, then filter by a query string

Every note is stored as a `.md.age` file (age-encrypted Markdown) inside a customer-specific subdirectory.

## Code Deep Dive

### The NoteStore Struct

```go
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
```

The store holds three pieces of state:

- **`baseDir`** — the root notes directory (e.g., `~/notes`)
- **`identity`** — the post-quantum key pair for encryption/decryption
- **`dateFormat`** — "EU" or "US", pulled from config

All three are injected via the constructor, keeping the store testable and free of global state.

### Date Layout Conversion

A helper converts the user-friendly format string to Go's time layout:

```go
func dateLayout(format string) string {
    if strings.ToUpper(format) == "US" {
        return "01-02-2006 15:04"
    }
    return "02-01-2006 15:04"
}
```

This centralized conversion ensures consistent date handling across all store operations.

### Creating a Note

The `Create` method generates a note file from a `Note` struct:

```go
func (s *NoteStore) Create(note *Note) (string, error) {
    layout := dateLayout(s.dateFormat)
    content := GenerateTemplate(note, layout)

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
```

The creation flow:

1. **Generate** the Markdown content with frontmatter using `GenerateTemplate`
2. **Sanitize** the customer name for use as a directory name
3. **Create** the customer directory if it doesn't exist
4. **Build** a date-prefixed filename from the note title
5. **Encrypt** the content and write it to the file
6. **Return** the full path of the created file

The method uses `s.identity.Recipient()` to get the public key for encryption. This is the "encrypt to yourself" pattern — the same key pair that creates the note can decrypt it later.

### Reading a Note

Reading reverses the creation process:

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

1. **Decrypt** the `.md.age` file using the identity (private key)
2. **Parse** the plaintext into a Note struct
3. **Set** the `FilePath` so the caller knows where the note came from

### Updating a Note

The update operation handles two cases — notes with custom body content and notes that should use a fresh template:

```go
func (s *NoteStore) Update(path string, note *Note) error {
    layout := dateLayout(s.dateFormat)

    var content string
    if note.Body != "" {
        content = renderWithBody(note, layout)
    } else {
        content = GenerateTemplate(note, layout)
    }

    if err := crypto.EncryptToFile(path, []byte(content), s.identity.Recipient()); err != nil {
        return fmt.Errorf("failed to encrypt updated note: %w", err)
    }

    return nil
}
```

The `renderWithBody` helper regenerates the frontmatter (to capture any metadata changes) but preserves the user's existing body text:

```go
func renderWithBody(note *Note, dateFormat string) string {
    var sb strings.Builder

    fm := frontmatterOut{
        Customer: note.Customer,
        Type:     note.Type.String(),
        Created:  formatDate(note.Created, dateFormat),
        Status:   string(note.Status),
    }
    // ... optional fields ...

    frontmatterBytes, _ := yaml.Marshal(&fm)

    sb.WriteString("---\n")
    sb.Write(frontmatterBytes)
    sb.WriteString("---\n\n")
    sb.WriteString(note.Body)

    return sb.String()
}
```

This separation is important: if you change a note's status from "open" to "done", you want the frontmatter updated but the body (your meeting notes, task description, etc.) preserved exactly as you wrote it.

### Listing All Notes

The `List` method walks the entire directory tree to find and decrypt all notes:

```go
func (s *NoteStore) List() ([]*Note, error) {
    var notes []*Note

    err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if info.IsDir() && info.Name() == ".pq-notes" {
            return filepath.SkipDir
        }

        if info.IsDir() || !strings.HasSuffix(info.Name(), ".md.age") {
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
        return nil, fmt.Errorf("failed to walk notes directory: %w", err)
    }

    sort.Slice(notes, func(i, j int) bool {
        return notes[i].Created.After(notes[j].Created)
    })

    return notes, nil
}
```

Design decisions:

- **`filepath.Walk`** recursively traverses the directory tree, calling the callback for every file and directory.
- **Skip `.pq-notes`** — the config directory is inside the notes directory, so we skip it with `filepath.SkipDir`.
- **Silently skip unreadable files** — if a file can't be decrypted or parsed, the walk continues. This is a graceful degradation choice: one corrupted file shouldn't prevent listing all other notes.
- **Sort by creation date** (newest first) — the most recent notes are typically the most relevant.

### Searching Notes

Search builds on List, adding a filter:

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
```

The search is a simple case-insensitive substring match across four fields: title, customer, body, and tags. This approach decrypts every note for each search — acceptable for a personal notes tool, though a larger system would want an index.

The `matchesQuery` function uses early returns: as soon as any field matches, it returns `true` without checking the remaining fields.

## Relationships

- **Crypto** — called for every file operation (encrypt on write, decrypt on read)
- **Note** — model parsing (`ParseNote`) and template generation (`GenerateTemplate`, `renderWithBody`)
- **Config** — provides the `dateFormat` setting and notes directory path
- **Editor** — works alongside NoteStore in the edit workflow (NoteStore decrypts, Editor opens, NoteStore re-encrypts)

The NoteStore is the most connected component in the architecture — it pulls together all the lower-level packages into a coherent API.

## Key Takeaways

- **`filepath.Walk`** is Go's built-in recursive directory traversal — use the callback to filter and process files.
- **`filepath.SkipDir`** is a special error value that tells Walk to skip an entire directory subtree.
- **Graceful degradation** (silently skipping bad files during listing) is appropriate for resilience, though you should log these failures in production.
- **Separating metadata updates from body updates** (`renderWithBody` vs `GenerateTemplate`) preserves user content while allowing metadata changes.
- **Constructor injection** of dependencies (identity, baseDir, dateFormat) keeps the store testable and decoupled from global state.
- **`sort.Slice`** with a comparison function is the idiomatic way to sort slices in Go.

## Next Steps

We've built the core CRUD layer. In the next chapter, we'll add business calendar support — determining which days are workdays and calculating future deadlines that respect weekends and holidays.
