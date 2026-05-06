package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func Install(binaryPath string) error {
	switch runtime.GOOS {
	case "linux":
		return installSystemd(binaryPath)
	case "darwin":
		return installLaunchd(binaryPath)
	case "windows":
		return installScheduledTask(binaryPath)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallSystemd()
	case "darwin":
		return uninstallLaunchd()
	case "windows":
		return uninstallScheduledTask()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func Status() (string, error) {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("systemctl", "--user", "is-active", "pq-notes").Output()
		return string(out), err
	default:
		return "unknown", nil
	}
}

func installSystemd(binaryPath string) error {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	os.MkdirAll(dir, 0755)
	unit := fmt.Sprintf("[Unit]\nDescription=pq-notes notification daemon\n\n[Service]\nExecStart=%s daemon\nRestart=always\n\n[Install]\nWantedBy=default.target\n", binaryPath)
	if err := os.WriteFile(filepath.Join(dir, "pq-notes.service"), []byte(unit), 0644); err != nil {
		return err
	}
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return exec.Command("systemctl", "--user", "enable", "--now", "pq-notes").Run()
}

func uninstallSystemd() error {
	exec.Command("systemctl", "--user", "disable", "--now", "pq-notes").Run()
	path := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "pq-notes.service")
	os.Remove(path)
	return exec.Command("systemctl", "--user", "daemon-reload").Run()
}

func installLaunchd(binaryPath string) error {
	plist := fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\">\n<dict>\n    <key>Label</key><string>com.pq-notes.daemon</string>\n    <key>ProgramArguments</key><array><string>%s</string><string>daemon</string></array>\n    <key>RunAtLoad</key><true/>\n    <key>KeepAlive</key><true/>\n</dict>\n</plist>", binaryPath)
	path := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.pq-notes.daemon.plist")
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", path).Run()
}

func uninstallLaunchd() error {
	path := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.pq-notes.daemon.plist")
	exec.Command("launchctl", "unload", path).Run()
	return os.Remove(path)
}

func installScheduledTask(binaryPath string) error {
	return exec.Command("schtasks", "/create", "/sc", "onlogon", "/tn", "pq-notes-daemon",
		"/tr", binaryPath+" daemon", "/f").Run()
}

func uninstallScheduledTask() error {
	return exec.Command("schtasks", "/delete", "/tn", "pq-notes-daemon", "/f").Run()
}
