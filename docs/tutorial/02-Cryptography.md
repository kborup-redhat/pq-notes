---
title: "Chapter 2: Post-Quantum Cryptography"
order: 2
---

# Chapter 2: Post-Quantum Cryptography

## Introduction

Imagine you have a safe in your house. Today's safes use mechanical locks that are practically impossible to pick with current tools. But what if someone invented a new kind of tool -- one that could crack any mechanical lock in seconds? You would need a fundamentally different kind of safe.

That is the situation with cryptography and quantum computers. Most encryption used today (RSA, standard elliptic curves) can be broken by a sufficiently powerful quantum computer running Shor's algorithm. While large-scale quantum computers do not exist yet, encrypted data stolen today could be decrypted years from now when they do -- a strategy known as "harvest now, decrypt later."

pq-notes addresses this by using **hybrid encryption**: it combines the proven X25519 elliptic curve algorithm with ML-KEM-768 (formerly CRYSTALS-Kyber), a post-quantum key encapsulation mechanism standardized by NIST. If either algorithm holds, your data stays safe. This is defense in depth for cryptography.

## How It Works

The crypto layer in pq-notes is built on top of [filippo.io/age](https://github.com/FiloSottile/age), a modern encryption library by Filippo Valsorda (the former Go team cryptography lead). The `age` library handles all the heavy lifting: key generation, key serialization, encryption, and decryption. pq-notes wraps it in a thin API tailored to the application's needs.

The encryption flow works like this:

1. **Key generation:** On first run, pq-notes generates a `HybridIdentity` -- a key pair that combines X25519 and ML-KEM-768. The private key (identity) is saved to `key.txt` in the config directory.
2. **Encryption:** When saving a note, the plaintext markdown is encrypted using the public key (recipient) derived from the identity. The ciphertext is written to disk as a `.md.age` file.
3. **Decryption:** When opening a note, the `.md.age` file is read and decrypted using the private identity.
4. **Sharing:** To share a note, it is re-encrypted for the recipient's public key. The recipient decrypts it with their own private identity.

The identity (private key) never leaves the local machine. Drive sync uploads only encrypted files. Even if someone intercepts the sync or steals the Drive folder, they cannot read the notes without the identity file.

## Code Deep Dive

All cryptography code lives in `internal/crypto/crypto.go`. Let's walk through each function.

### Key Generation

The `GenerateKey` function creates a new hybrid key pair and saves the private identity to a file:

```go
// GenerateKey generates a post-quantum hybrid key pair using age.GenerateHybridIdentity()
// and writes the identity string to keyPath with 0600 permissions.
func GenerateKey(keyPath string) (*age.HybridIdentity, error) {
    identity, err := age.GenerateHybridIdentity()
    if err != nil {
        return nil, fmt.Errorf("failed to generate hybrid identity: %w", err)
    }

    // Write the identity to file with 0600 permissions
    identityStr := identity.String()
    err = os.WriteFile(keyPath, []byte(identityStr+"\n"), 0600)
    if err != nil {
        return nil, fmt.Errorf("failed to write key to file: %w", err)
    }

    return identity, nil
}
```

A few important details:

- `age.GenerateHybridIdentity()` is where the magic happens. It generates both an X25519 key pair and an ML-KEM-768 key pair, then combines them into a single identity. The "hybrid" means both algorithms must be broken to compromise the key.
- The identity is written with `0600` permissions -- only the file owner can read or write it. This is your private key; protecting it is critical.
- The `%w` verb in `fmt.Errorf` wraps the original error, preserving the error chain for callers who want to inspect the root cause with `errors.Is` or `errors.As`.

### Loading an Existing Identity

On subsequent launches, pq-notes loads the identity from disk rather than generating a new one:

```go
// LoadIdentity reads a key file, parses identities with age.ParseIdentities(),
// and returns the first *age.HybridIdentity found.
func LoadIdentity(keyPath string) (*age.HybridIdentity, error) {
    file, err := os.Open(keyPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open key file: %w", err)
    }
    defer file.Close()

    identities, err := age.ParseIdentities(file)
    if err != nil {
        return nil, fmt.Errorf("failed to parse identities: %w", err)
    }

    if len(identities) == 0 {
        return nil, fmt.Errorf("no identities found in key file")
    }

    // Find the first HybridIdentity
    for _, id := range identities {
        if hybrid, ok := id.(*age.HybridIdentity); ok {
            return hybrid, nil
        }
    }

    return nil, fmt.Errorf("no HybridIdentity found in key file")
}
```

This function demonstrates a common Go pattern: **type assertion with a range loop**. The `age.ParseIdentities` function returns a slice of `age.Identity` interfaces. Since a key file could theoretically contain multiple identities of different types, we iterate and use `id.(*age.HybridIdentity)` to find the specific type we need. The `ok` variable in the two-value type assertion prevents panics if the assertion fails.

Also note the `defer file.Close()` -- a Go idiom that ensures the file handle is released when the function returns, regardless of whether it returns normally or via an error.

### Extracting the Public Key

The public key is what you share with others so they can encrypt notes for you:

```go
// PublicKey returns the public key string from a HybridIdentity.
func PublicKey(identity *age.HybridIdentity) string {
    return identity.Recipient().String()
}
```

In age's terminology, a **Recipient** is the public half of a key pair -- the entity that can receive encrypted messages. An **Identity** is the private half -- the entity that can decrypt them. `Recipient().String()` serializes the public key to a string that starts with `age1` and can be safely shared.

### In-Memory Encryption

The `Encrypt` function encrypts plaintext bytes in memory without touching the filesystem:

```go
// Encrypt encrypts plaintext in memory using age.Encrypt.
func Encrypt(plaintext []byte, recipients ...age.Recipient) ([]byte, error) {
    if len(recipients) == 0 {
        return nil, fmt.Errorf("no recipients provided")
    }

    var buf bytes.Buffer
    w, err := age.Encrypt(&buf, recipients...)
    if err != nil {
        return nil, fmt.Errorf("failed to create encryptor: %w", err)
    }

    _, err = w.Write(plaintext)
    if err != nil {
        return nil, fmt.Errorf("failed to write plaintext: %w", err)
    }

    err = w.Close()
    if err != nil {
        return nil, fmt.Errorf("failed to finalize encryption: %w", err)
    }

    return buf.Bytes(), nil
}
```

The variadic `recipients ...age.Recipient` parameter is key to the sharing feature. When encrypting a note for yourself, you pass your own recipient (public key). When sharing, you pass both your recipient and the contact's recipient -- the note can then be decrypted by either party.

The `age.Encrypt` function returns a `WriteCloser`. You must call `Close()` to finalize the encryption -- without it, the ciphertext is incomplete. This is a common pattern in streaming encryption APIs.

### In-Memory Decryption

Decryption is the inverse:

```go
// Decrypt decrypts ciphertext in memory using age.Decrypt.
func Decrypt(ciphertext []byte, identities ...age.Identity) ([]byte, error) {
    if len(identities) == 0 {
        return nil, fmt.Errorf("no identities provided")
    }

    r, err := age.Decrypt(bytes.NewReader(ciphertext), identities...)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt: %w", err)
    }

    plaintext, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("failed to read decrypted data: %w", err)
    }

    return plaintext, nil
}
```

Where encryption uses a `WriteCloser`, decryption uses a `Reader`. `io.ReadAll` drains the reader into a byte slice. The variadic `identities` parameter allows passing multiple identities -- age will try each one until it finds one that can decrypt the message.

### File-Based Encryption

For writing encrypted notes directly to disk, `EncryptToFile` streams the ciphertext to a file rather than buffering it in memory:

```go
// EncryptToFile encrypts plaintext and writes it to a file.
func EncryptToFile(path string, plaintext []byte, recipients ...age.Recipient) error {
    if len(recipients) == 0 {
        return fmt.Errorf("no recipients provided")
    }

    file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()

    w, err := age.Encrypt(file, recipients...)
    if err != nil {
        return fmt.Errorf("failed to create encryptor: %w", err)
    }

    _, err = w.Write(plaintext)
    if err != nil {
        return fmt.Errorf("failed to write plaintext: %w", err)
    }

    err = w.Close()
    if err != nil {
        return fmt.Errorf("failed to finalize encryption: %w", err)
    }

    return nil
}
```

The file flags `os.O_WRONLY|os.O_CREATE|os.O_TRUNC` mean: open for writing only, create it if it does not exist, and truncate (overwrite) it if it does. Combined with `0600` permissions, this ensures secure file handling.

### File-Based Decryption

Reading and decrypting a file from disk:

```go
// DecryptFile reads and decrypts a file.
func DecryptFile(path string, identities ...age.Identity) ([]byte, error) {
    if len(identities) == 0 {
        return nil, fmt.Errorf("no identities provided")
    }

    file, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    r, err := age.Decrypt(file, identities...)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt: %w", err)
    }

    plaintext, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("failed to read decrypted data: %w", err)
    }

    return plaintext, nil
}
```

This is the function called every time you open a note in the TUI. The NoteStore calls `DecryptFile`, gets back plaintext markdown, passes it to `ParseNote` (Chapter 3) to extract frontmatter and body, and renders the markdown in the preview pane.

## Relationships to Other Components

- **Configuration (Chapter 1):** The key file path is derived from the config directory: `ConfigDirIn(notesDir) + "/key.txt"`. The config directory's `0700` permissions provide an additional layer of protection for the key.
- **Note Model (Chapter 3):** Notes are stored as `.md.age` files. The `NoteFilename` function generates filenames ending in `.md.age` to indicate encrypted markdown.
- **NoteStore:** The NoteStore calls `EncryptToFile` when saving and `DecryptFile` when loading notes, making encryption transparent to the rest of the application.
- **Sharing:** When sharing a note, the NoteStore decrypts it with the owner's identity, then re-encrypts it with both the owner's and the contact's recipients. The shared file gets a `.md.age.shared` extension.
- **Drive Sync:** Only encrypted `.md.age` files are uploaded to Drive. The plaintext never leaves the local machine.

## Key Takeaways

- pq-notes uses **hybrid encryption** (X25519 + ML-KEM-768) that resists both classical and quantum attacks.
- The `age` library provides the entire cryptographic foundation. pq-notes adds a thin wrapper for its specific needs (key file management, file-based operations).
- The variadic `recipients`/`identities` parameters enable note sharing: encrypt once for multiple recipients.
- Every file operation uses restrictive permissions (`0600`) because the data is sensitive by nature.
- The `WriteCloser` pattern in encryption means you **must** call `Close()` to produce valid ciphertext.

## Next Steps

We now have configuration and encryption in place. In [Chapter 3: The Note Model](03-Note-Model.md), we will see how pq-notes structures notes as markdown files with YAML frontmatter, supports multiple note types, and generates type-specific templates.
