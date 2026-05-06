---
title: "Chapter 2: Crypto"
order: 2
---

# Chapter 2: Post-Quantum Encryption

Security is the cornerstone of pq-notes. Every note is encrypted before it touches the disk, and the encryption uses **post-quantum hybrid cryptography** — a combination of classical and quantum-resistant algorithms. In this chapter, we'll build the encryption layer that makes this possible.

Think of this package as a locksmith that creates special locks (key pairs) and can lock (encrypt) or unlock (decrypt) any note.

## How It Works

The crypto package wraps the `filippo.io/age` library to provide six operations:

1. **GenerateKey** — Create a new post-quantum hybrid key pair
2. **LoadIdentity** — Read an existing key from a file
3. **PublicKey** — Extract the public key from an identity
4. **Encrypt / Decrypt** — In-memory encryption and decryption
5. **EncryptToFile / DecryptFile** — File-based encryption and decryption

### What is Post-Quantum Hybrid Encryption?

Traditional public-key encryption (like RSA or X25519) relies on mathematical problems that quantum computers could eventually solve. **Post-quantum** algorithms use different math that remains hard even for quantum computers.

A **hybrid** approach combines both:
- **ML-KEM** (Module Lattice Key Encapsulation Mechanism) — quantum-resistant
- **X25519** — battle-tested classical elliptic curve algorithm

If either algorithm holds, your data stays safe. This gives you quantum resistance today while keeping the proven security of classical algorithms as a safety net.

## Code Deep Dive

### Key Generation

The `GenerateKey` function creates a new hybrid identity and saves it to disk:

```go
func GenerateKey(keyPath string) (*age.HybridIdentity, error) {
    identity, err := age.GenerateHybridIdentity()
    if err != nil {
        return nil, fmt.Errorf("failed to generate hybrid identity: %w", err)
    }

    identityStr := identity.String()
    err = os.WriteFile(keyPath, []byte(identityStr+"\n"), 0600)
    if err != nil {
        return nil, fmt.Errorf("failed to write key to file: %w", err)
    }

    return identity, nil
}
```

Key details:

- **`age.GenerateHybridIdentity()`** generates a random ML-KEM + X25519 key pair in a single call.
- The identity is serialized to a string and written to a file with **`0600` permissions** — only the owner can read or write the key file. This is critical: the identity file is the private key. If someone else can read it, they can decrypt all your notes.
- **`%w` in `fmt.Errorf`** wraps the original error, preserving the full error chain for debugging. Callers can use `errors.Is()` or `errors.As()` to inspect the underlying cause.
- The trailing `\n` ensures the file ends with a newline, which is a convention that many tools expect.

### Loading an Existing Key

When the application starts, it needs to load the key that was previously generated:

```go
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

    for _, id := range identities {
        if hybrid, ok := id.(*age.HybridIdentity); ok {
            return hybrid, nil
        }
    }

    return nil, fmt.Errorf("no HybridIdentity found in key file")
}
```

This function demonstrates several important Go patterns:

- **`defer file.Close()`** ensures the file is closed when the function returns, regardless of which return path is taken. This prevents resource leaks.
- **Type assertion** (`id.(*age.HybridIdentity)`) safely checks whether an interface value holds a specific concrete type. The two-value form (`hybrid, ok`) prevents a panic if the type doesn't match.
- The function iterates through all identities in the file looking for a `HybridIdentity`, since the age format supports multiple identity types.

### Extracting the Public Key

The public key is what you share with others (or use to encrypt to yourself):

```go
func PublicKey(identity *age.HybridIdentity) string {
    return identity.Recipient().String()
}
```

In age's terminology, a **Recipient** is the public half of a key pair — the thing you encrypt *to*. An **Identity** is the private half — the thing you decrypt *with*.

### In-Memory Encryption

The `Encrypt` function encrypts data without touching the filesystem:

```go
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

The encryption process follows a **streaming writer** pattern:

1. Create an `age.Encrypt` writer that wraps a buffer
2. Write plaintext into the writer
3. **Close the writer** — this is essential, as it finalizes the encryption and writes any remaining buffered data

The `...age.Recipient` variadic parameter means you can encrypt to multiple recipients. Each recipient gets their own copy of the data encryption key, wrapped with their public key.

### In-Memory Decryption

Decryption mirrors encryption with a reader pattern:

```go
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

The age library tries each identity until one successfully decrypts the file header. If none match, it returns an error.

### File-Based Operations

For notes stored on disk, `EncryptToFile` and `DecryptFile` handle the file I/O:

```go
func EncryptToFile(path string, plaintext []byte, recipients ...age.Recipient) error {
    file, err := os.Create(path)
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

    return w.Close()
}
```

Notice how `EncryptToFile` writes directly to the file (via `os.Create`) instead of buffering in memory first. This is more efficient for large notes, though in practice notes are small enough that the difference is negligible.

`DecryptFile` reads from a file and returns the decrypted plaintext:

```go
func DecryptFile(path string, identities ...age.Identity) ([]byte, error) {
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

## Relationships

- **NoteStore** is the primary consumer — it calls `EncryptToFile` when creating/updating notes and `DecryptFile` when reading them.
- **Config** stores the path to the key file (inside the `.pq-notes` directory), which is passed to `LoadIdentity` at startup.

The crypto package itself has no dependencies on other internal packages. It depends only on `filippo.io/age` and Go's standard library.

## Key Takeaways

- **Post-quantum hybrid encryption** combines ML-KEM (quantum-resistant) and X25519 (classical) for defense-in-depth.
- **`defer`** is essential for resource cleanup — always close files, network connections, and similar resources.
- **File permissions matter** — private keys should be `0600` (owner read/write only).
- **Error wrapping** with `%w` preserves the error chain, making debugging easier.
- **Streaming I/O** (writers for encryption, readers for decryption) is a common Go pattern that keeps memory usage predictable.
- **Always `Close()` encryption writers** — failing to close means incomplete or corrupted output.

## Next Steps

We now have a way to securely store data. In the next chapter, we'll define what that data looks like — the Note model with its YAML frontmatter and type-specific Markdown templates.
