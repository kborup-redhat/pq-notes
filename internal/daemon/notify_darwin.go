//go:build darwin

package daemon

import "os/exec"

func sendOSNotification(title, body string) error {
	script := `display notification "` + body + `" with title "` + title + `"`
	return exec.Command("osascript", "-e", script).Run()
}
