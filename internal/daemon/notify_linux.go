//go:build linux

package daemon

import "os/exec"

func sendOSNotification(title, body string) error {
	return exec.Command("notify-send", title, body).Run()
}
