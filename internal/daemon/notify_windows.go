//go:build windows

package daemon

import "os/exec"

func sendOSNotification(title, body string) error {
	script := `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; ` +
		`$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); ` +
		`$textNodes = $template.GetElementsByTagName('text'); ` +
		`$textNodes.Item(0).AppendChild($template.CreateTextNode('` + title + `')) > $null; ` +
		`$textNodes.Item(1).AppendChild($template.CreateTextNode('` + body + `')) > $null; ` +
		`$toast = [Windows.UI.Notifications.ToastNotification]::new($template); ` +
		`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('pq-notes').Show($toast)`
	return exec.Command("powershell", "-Command", script).Run()
}
