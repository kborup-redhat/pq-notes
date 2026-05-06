# Installation Guide

## Download pre-built binaries

Download the latest release for your platform from the [releases page](https://github.com/kborup-redhat/pq-notes/releases).

| Platform | Binary |
|----------|--------|
| Linux (x86_64) | `pq-notes-linux-amd64` |
| Linux (ARM64) | `pq-notes-linux-arm64` |
| macOS (Intel) | `pq-notes-darwin-amd64` |
| macOS (Apple Silicon) | `pq-notes-darwin-arm64` |
| Windows (x86_64) | `pq-notes-windows-amd64.exe` |

## Linux

### Option 1: Download binary

```bash
# Download (replace with your architecture)
curl -L -o pq-notes https://github.com/kborup-redhat/pq-notes/releases/latest/download/pq-notes-linux-amd64

# Make executable
chmod +x pq-notes

# Move to PATH
sudo mv pq-notes /usr/local/bin/

# Verify
pq-notes --help
```

### Option 2: Build from source

```bash
# Requires Go 1.26+
git clone https://github.com/kborup-redhat/pq-notes.git
cd pq-notes
go build -o pq-notes .
sudo mv pq-notes /usr/local/bin/
```

### Option 3: Go install

```bash
go install github.com/kborup-redhat/pq-notes@latest
```

### Set up notifications (optional)

```bash
# Install as systemd user service
pq-notes daemon install

# Check status
pq-notes daemon status

# Remove
pq-notes daemon uninstall
```

## macOS

### Option 1: Download binary

```bash
# Apple Silicon (M1/M2/M3/M4)
curl -L -o pq-notes https://github.com/kborup-redhat/pq-notes/releases/latest/download/pq-notes-darwin-arm64

# Intel Mac
curl -L -o pq-notes https://github.com/kborup-redhat/pq-notes/releases/latest/download/pq-notes-darwin-amd64

# Make executable
chmod +x pq-notes

# Move to PATH
sudo mv pq-notes /usr/local/bin/

# macOS may block unsigned binaries — allow it:
# System Settings > Privacy & Security > Allow "pq-notes"
# Or run:
xattr -d com.apple.quarantine /usr/local/bin/pq-notes
```

### Option 2: Build from source

```bash
git clone https://github.com/kborup-redhat/pq-notes.git
cd pq-notes
go build -o pq-notes .
sudo mv pq-notes /usr/local/bin/
```

### Set up notifications (optional)

```bash
# Install as launchd agent (runs at login)
pq-notes daemon install

# Check status
pq-notes daemon status

# Remove
pq-notes daemon uninstall
```

## Windows

### Option 1: Download binary

1. Download `pq-notes-windows-amd64.exe` from the [releases page](https://github.com/kborup-redhat/pq-notes/releases)
2. Rename to `pq-notes.exe` (optional)
3. Move to a directory in your PATH, or add its location to PATH:
   - Open Settings > System > About > Advanced system settings
   - Click "Environment Variables"
   - Edit the `Path` variable and add the directory containing `pq-notes.exe`
4. Open a new terminal and run: `pq-notes --help`

### Option 2: Build from source

```powershell
git clone https://github.com/kborup-redhat/pq-notes.git
cd pq-notes
go build -o pq-notes.exe .
```

### Set up notifications (optional)

```powershell
# Install as scheduled task (runs at logon)
pq-notes daemon install

# Check status
pq-notes daemon status

# Remove
pq-notes daemon uninstall
```

## First run

On first launch, pq-notes runs a setup wizard:

```bash
pq-notes
```

The wizard will ask you to:

1. **Choose an editor** — vim, nano, or VS Code (or type a custom command)
2. **Select a date format** — e.g., `2006-01-02`, `02/01/2006`, `Jan 2, 2006`
3. **Set your country** — Determines public holidays and business days
4. **Configure weekend days** — Defaults based on country (Sat/Sun or Fri/Sat)
5. **Generate encryption key** — Creates your age hybrid key pair

Your notes directory will be created at `~/notes/` with the configuration stored in `~/notes/.pq-notes/`.

## Google Drive setup (optional)

To back up encrypted notes to Google Drive:

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create a project (or select an existing one)
3. Enable the Google Drive API
4. Create an OAuth 2.0 Client ID (application type: Desktop)
5. Download the JSON credentials file
6. Run:

```bash
pq-notes drive setup
```

7. Paste the path to the downloaded credentials file when prompted
8. Complete the browser sign-in flow
9. Enable auto-sync (optional):

```bash
pq-notes drive auto
```

## Migrating to a new machine

1. Install pq-notes on the new machine
2. Copy your key file from the old machine:
   ```bash
   # On old machine
   pq-notes key export --private
   
   # On new machine
   pq-notes key import <key-file>
   ```
3. Set up Drive and pull your notes:
   ```bash
   pq-notes drive setup
   pq-notes drive pull
   ```

## Shell completions

```bash
# Bash
pq-notes completion bash > /etc/bash_completion.d/pq-notes

# Zsh
pq-notes completion zsh > "${fpath[1]}/_pq-notes"

# Fish
pq-notes completion fish > ~/.config/fish/completions/pq-notes.fish

# PowerShell
pq-notes completion powershell > pq-notes.ps1
```

## Uninstalling

```bash
# Remove daemon if installed
pq-notes daemon uninstall

# Remove binary
sudo rm /usr/local/bin/pq-notes

# Remove notes and config (careful — this deletes all your notes!)
# rm -rf ~/notes/
```
